// Package filebase implements the warm-tier (CLAUDE.md §4.5) tiers.Client
// against Filebase's S3-compatible API. Filebase returns the IPFS CID in
// the x-amz-meta-cid header on PutObject, so the warm-tier blob is
// addressable by either its S3 key or the CID (Filebase exposes both).
//
// Key derivation: the S3 object key is sha256(data) hex-encoded. This
// keeps Puts idempotent/dedupable and avoids relying on random keys
// that the caller would have to persist separately from the CID.
package filebase

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	smithymw "github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"

	"github.com/isongjosiah/lbvr-med/internal/config"
	"github.com/isongjosiah/lbvr-med/internal/tiers"
)

const (
	// defaultEndpoint is the Filebase S3 endpoint. Kept exported-via-Option
	// for tests; production code never overrides it.
	defaultEndpoint = "https://s3.filebase.com"
	defaultRegion   = "us-east-1"
	defaultTimeout  = 30 * time.Second

	// MaxUploadBytes matches the Pinata cap for consistency with §4.5.
	MaxUploadBytes = 128 * 1024 * 1024

	// cidMetaHeader is Filebase's non-standard S3 response header that
	// carries the IPFS CID on successful PutObject.
	cidMetaHeader = "x-amz-meta-cid"
)

// Client is the Filebase warm-tier client.
type Client struct {
	s3          *s3.Client
	httpClient  *http.Client
	gatewayBase string // https://<bucket>.s3.filebase.com
	bucket      string
	endpoint    string
}

// Option customises a Client.
type Option func(*Client)

// WithEndpoint overrides the S3 endpoint. Used by tests.
func WithEndpoint(u string) Option { return func(c *Client) { c.endpoint = u } }

// WithGatewayBase overrides the CID-gateway URL used by Get. Used by tests
// to redirect CID fetches at httptest.
func WithGatewayBase(u string) Option { return func(c *Client) { c.gatewayBase = u } }

// WithHTTPClient overrides the HTTP client used for CID-gateway fetches.
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.httpClient = h } }

// WithS3Client injects a pre-built *s3.Client. Used by tests to supply
// a client whose BaseEndpoint and HTTPClient point at httptest.
func WithS3Client(s *s3.Client) Option { return func(c *Client) { c.s3 = s } }

// New constructs a Filebase Client. It errors if any of the three required
// env vars is empty — the warm tier has no fallback credentials.
func New(cfg *config.Config, opts ...Option) (*Client, error) {
	if cfg == nil {
		return nil, errors.New("filebase: nil config")
	}
	if cfg.FilebaseAccessKey == "" {
		return nil, errors.New("filebase: FILEBASE_ACCESS_KEY is empty")
	}
	if cfg.FilebaseSecretKey == "" {
		return nil, errors.New("filebase: FILEBASE_SECRET_KEY is empty")
	}
	if cfg.FilebaseBucket == "" {
		return nil, errors.New("filebase: FILEBASE_BUCKET is empty")
	}

	c := &Client{
		httpClient:  &http.Client{Timeout: defaultTimeout},
		bucket:      cfg.FilebaseBucket,
		endpoint:    defaultEndpoint,
		gatewayBase: fmt.Sprintf("https://%s.s3.filebase.com", cfg.FilebaseBucket),
	}
	for _, o := range opts {
		o(c)
	}

	if c.s3 == nil {
		awsCfg := aws.Config{
			Region: defaultRegion,
			Credentials: credentials.NewStaticCredentialsProvider(
				cfg.FilebaseAccessKey, cfg.FilebaseSecretKey, "",
			),
		}
		c.s3 = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(c.endpoint)
			// Filebase requires path-style addressing (no bucket vhost).
			o.UsePathStyle = true
		})
	}
	return c, nil
}

// Name implements tiers.Client.
func (c *Client) Name() string { return "filebase" }

// TierClass implements tiers.Client.
func (c *Client) TierClass() uint8 { return tiers.TierWarm }

// keyFor returns the S3 object key for data. Deterministic so re-uploads
// of identical bundles dedupe to the same key — Filebase itself still
// returns the same CID for identical content.
func keyFor(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// Put uploads data, returning the Filebase-reported IPFS CID from the
// x-amz-meta-cid response header. Returns an error if the header is
// absent — without the CID the object is not retrievable via the
// rest of the LBVR-Med stack (which addresses by CID, not S3 key).
func (c *Client) Put(ctx context.Context, data []byte) (string, error) {
	if len(data) == 0 {
		return "", errors.New("filebase: empty payload")
	}
	if int64(len(data)) > MaxUploadBytes {
		return "", fmt.Errorf("filebase: payload %d bytes exceeds MaxUploadBytes %d", len(data), MaxUploadBytes)
	}

	key := keyFor(data)

	// We tap the raw HTTP response via a Deserialize-middleware because
	// x-amz-meta-cid is only reliably available on the raw response; some
	// SDK minor versions fail to hoist it into out.Metadata. Reading the
	// header directly is authoritative.
	var capturedCID string
	capture := &cidHeaderTap{dst: &capturedCID}

	out, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	}, func(o *s3.Options) {
		o.APIOptions = append(o.APIOptions, func(stack *smithymw.Stack) error {
			return stack.Deserialize.Add(capture, smithymw.After)
		})
	})
	if err != nil {
		// err text is SDK-generated; it never includes our secrets.
		return "", fmt.Errorf("filebase: put: %w", err)
	}
	if out == nil {
		return "", errors.New("filebase: nil PutObject response")
	}
	if capturedCID != "" {
		return capturedCID, nil
	}
	// s3.PutObjectOutput has no Metadata field (unlike GetObjectOutput);
	// the middleware capture above is the only path. If it did not fire,
	// the response genuinely lacked the header — Filebase misconfigured.
	return "", errors.New("filebase: PutObject response missing x-amz-meta-cid")
}

// Get fetches the object by CID through Filebase's IPFS gateway URL
// (see CLAUDE.md §11 — no public ipfs.io). The bucket's own S3 URL
// supports /ipfs/<cid> for pinned content.
func (c *Client) Get(ctx context.Context, cid string) ([]byte, error) {
	if cid == "" {
		return nil, errors.New("filebase: empty cid")
	}
	u := strings.TrimRight(c.gatewayBase, "/") + "/ipfs/" + cid

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("filebase: req: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("filebase: http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("filebase: get status=%d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, MaxUploadBytes+1))
}

// getByKey is the S3-key-addressed variant. Kept internal; the retrieval
// gateway always uses CID. Retained for ops/debugging.
func (c *Client) getByKey(ctx context.Context, key string) ([]byte, error) { //nolint:unused
	out, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("filebase: get-by-key: %w", err)
	}
	defer out.Body.Close()
	return io.ReadAll(io.LimitReader(out.Body, MaxUploadBytes+1))
}

// Stat uses S3 HeadObject on the deterministic key. HEAD on the CID
// gateway is also possible but returns less metadata (no LastModified
// with timezone fidelity, no ContentLength on some paths); HeadObject
// is cheaper and more reliable for PoR scheduling.
func (c *Client) Stat(ctx context.Context, cid string) (*tiers.Stat, error) {
	if cid == "" {
		return nil, errors.New("filebase: empty cid")
	}
	// The caller typically does not know the S3 key — only the CID. For
	// Stat-by-CID we HEAD the gateway URL instead; this is the only path
	// that works without caller-supplied key material.
	u := strings.TrimRight(c.gatewayBase, "/") + "/ipfs/" + cid
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, u, nil)
	if err != nil {
		return nil, fmt.Errorf("filebase: req: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("filebase: http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("filebase: stat status=%d", resp.StatusCode)
	}
	st := &tiers.Stat{CID: cid, SizeBytes: resp.ContentLength}
	if lm := resp.Header.Get("Last-Modified"); lm != "" {
		if ts, err := time.Parse(http.TimeFormat, lm); err == nil {
			st.StoredAt = ts
		}
	}
	return st, nil
}

// Delete issues S3 DeleteObject against the CID-as-key. Filebase
// internally indexes objects by both S3 key and CID, so either works as
// an unpin signal when the bucket is configured as IPFS-pinning (the
// default for LBVR-Med). DeleteByKey is preferred when the ingest CLI
// has stored the derived S3 key alongside the CID.
func (c *Client) Delete(ctx context.Context, cid string) error {
	if cid == "" {
		return errors.New("filebase: empty cid")
	}
	_, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(cid),
	})
	if err != nil {
		return fmt.Errorf("filebase: delete: %w", err)
	}
	return nil
}

// DeleteByKey is the preferred path when the ingest CLI has stored the
// S3 key alongside the CID in the registry.
func (c *Client) DeleteByKey(ctx context.Context, key string) error {
	if key == "" {
		return errors.New("filebase: empty key")
	}
	_, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("filebase: delete-by-key: %w", err)
	}
	return nil
}

// KeyFor exposes the deterministic key derivation so the ingest CLI can
// compute and persist it at upload time without re-hashing.
func KeyFor(data []byte) string { return keyFor(data) }

// cidHeaderTap is a smithy-go Deserialize middleware that records the
// Filebase x-amz-meta-cid response header into dst.
type cidHeaderTap struct {
	dst *string
}

// ID implements smithy middleware.Middleware.
func (cidHeaderTap) ID() string { return "LbvrFilebaseCIDCapture" }

// HandleDeserialize runs after the SDK has deserialized the response, so
// out.RawResponse is the *smithyhttp.Response we need.
func (h *cidHeaderTap) HandleDeserialize(
	ctx context.Context,
	in smithymw.DeserializeInput,
	next smithymw.DeserializeHandler,
) (smithymw.DeserializeOutput, smithymw.Metadata, error) {
	out, meta, err := next.HandleDeserialize(ctx, in)
	if err != nil {
		return out, meta, err
	}
	if resp, ok := out.RawResponse.(*smithyhttp.Response); ok && resp != nil {
		if v := resp.Header.Get(cidMetaHeader); v != "" {
			*h.dst = v
		}
	}
	return out, meta, err
}
