// Package pinata implements the hot-tier (CLAUDE.md §4.5) tiers.Client
// against the Pinata dedicated-gateway REST API. Secrets are never logged
// — the JWT is masked as "***" in every diagnostic path.
package pinata

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/isongjosiah/lbvr-med/internal/config"
	"github.com/isongjosiah/lbvr-med/internal/tiers"
)

const (
	apiBase = "https://api.pinata.cloud"
	// MaxUploadBytes is a defensive cap. Measured P99 Synthea bundle is
	// ~38 MB (CLAUDE.md §4.5); Pinata advertises 10 GB. 128 MiB lands
	// well above the workload and well below the backend ceiling.
	MaxUploadBytes = 128 * 1024 * 1024

	defaultGateway = "https://gateway.pinata.cloud"
	defaultTimeout = 30 * time.Second
)

// Client is the Pinata hot-tier client.
type Client struct {
	httpClient *http.Client
	apiBase    string // overrideable for tests
	gateway    string
	jwt        string
}

// Option customises a Client. Only used by tests and by callers that
// inject a pre-built *http.Client (e.g. with observability middleware).
type Option func(*Client)

// WithHTTPClient overrides the default *http.Client.
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.httpClient = h } }

// WithAPIBase overrides the Pinata API base URL; used by tests to point
// at httptest.NewServer. Not exported via a production config path.
func WithAPIBase(u string) Option { return func(c *Client) { c.apiBase = u } }

// WithGateway overrides the IPFS gateway URL used by Get.
func WithGateway(u string) Option { return func(c *Client) { c.gateway = u } }

// New constructs a Pinata Client from cfg. Returns an error if the JWT is
// empty — the hot tier cannot function without authentication.
func New(cfg *config.Config, opts ...Option) (*Client, error) {
	if cfg == nil {
		return nil, errors.New("pinata: nil config")
	}
	if cfg.PinataJWT == "" {
		return nil, errors.New("pinata: PINATA_JWT is empty")
	}
	gw := cfg.PinataGateway
	if gw == "" {
		gw = defaultGateway
	}
	c := &Client{
		httpClient: &http.Client{Timeout: defaultTimeout},
		apiBase:    apiBase,
		gateway:    strings.TrimRight(gw, "/"),
		jwt:        cfg.PinataJWT,
	}
	for _, o := range opts {
		o(c)
	}
	return c, nil
}

// Name implements tiers.Client.
func (c *Client) Name() string { return "pinata" }

// TierClass implements tiers.Client.
func (c *Client) TierClass() uint8 { return tiers.TierHot }

// pinFileResponse is the subset of the pinFileToIPFS response we consume.
type pinFileResponse struct {
	IpfsHash  string `json:"IpfsHash"`
	PinSize   int64  `json:"PinSize"`
	Timestamp string `json:"Timestamp"`
}

// Put uploads data and returns the IPFS CID reported by Pinata. The
// upload size is bounded by MaxUploadBytes.
func (c *Client) Put(ctx context.Context, data []byte) (string, error) {
	if len(data) == 0 {
		return "", errors.New("pinata: empty payload")
	}
	if int64(len(data)) > MaxUploadBytes {
		return "", fmt.Errorf("pinata: payload %d bytes exceeds MaxUploadBytes %d", len(data), MaxUploadBytes)
	}

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	part, err := mw.CreateFormFile("file", "bundle.bin")
	if err != nil {
		return "", fmt.Errorf("pinata: form: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return "", fmt.Errorf("pinata: form write: %w", err)
	}
	if err := mw.Close(); err != nil {
		return "", fmt.Errorf("pinata: form close: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiBase+"/pinning/pinFileToIPFS", body)
	if err != nil {
		return "", fmt.Errorf("pinata: req: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.jwt)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("pinata: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", c.httpError("Put", resp)
	}

	var pr pinFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return "", fmt.Errorf("pinata: decode: %w", err)
	}
	if pr.IpfsHash == "" {
		return "", errors.New("pinata: response missing IpfsHash")
	}
	return pr.IpfsHash, nil
}

// Get fetches the bundle via the configured dedicated gateway. Public
// ipfs.io is NEVER used (CLAUDE.md §11).
func (c *Client) Get(ctx context.Context, cid string) ([]byte, error) {
	if cid == "" {
		return nil, errors.New("pinata: empty cid")
	}
	u := c.gateway + "/ipfs/" + url.PathEscape(cid)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("pinata: req: %w", err)
	}
	// Dedicated gateways often require the JWT as a query or bearer token.
	// Bearer works for both the default and custom domains; a non-auth'd
	// public gateway would simply ignore it.
	req.Header.Set("Authorization", "Bearer "+c.jwt)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pinata: http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, c.httpError("Get", resp)
	}
	return io.ReadAll(io.LimitReader(resp.Body, MaxUploadBytes+1))
}

// pinListResponse mirrors https://docs.pinata.cloud/api-reference/endpoint/list-files
type pinListResponse struct {
	Count int `json:"count"`
	Rows  []struct {
		IpfsPinHash string `json:"ipfs_pin_hash"`
		Size        int64  `json:"size"`
		DatePinned  string `json:"date_pinned"`
	} `json:"rows"`
}

// Stat returns metadata for the first pin whose hash matches cid.
func (c *Client) Stat(ctx context.Context, cid string) (*tiers.Stat, error) {
	if cid == "" {
		return nil, errors.New("pinata: empty cid")
	}
	q := url.Values{}
	q.Set("hashContains", cid)
	q.Set("status", "pinned")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiBase+"/data/pinList?"+q.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("pinata: req: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.jwt)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pinata: http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, c.httpError("Stat", resp)
	}

	var plr pinListResponse
	if err := json.NewDecoder(resp.Body).Decode(&plr); err != nil {
		return nil, fmt.Errorf("pinata: decode: %w", err)
	}
	if len(plr.Rows) == 0 {
		return nil, fmt.Errorf("pinata: cid %s not pinned", cid)
	}
	row := plr.Rows[0]
	ts, _ := time.Parse(time.RFC3339, row.DatePinned) // zero value is acceptable
	return &tiers.Stat{
		CID:       row.IpfsPinHash,
		SizeBytes: row.Size,
		StoredAt:  ts,
	}, nil
}

// Delete unpins the content. Note that unpinning does not evict from
// the public IPFS network immediately; it only removes Pinata's pin.
func (c *Client) Delete(ctx context.Context, cid string) error {
	if cid == "" {
		return errors.New("pinata: empty cid")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.apiBase+"/pinning/unpin/"+url.PathEscape(cid), nil)
	if err != nil {
		return fmt.Errorf("pinata: req: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.jwt)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("pinata: http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return c.httpError("Delete", resp)
	}
	return nil
}

// httpError builds an error that carries the response status without
// leaking any part of the JWT. The body is capped to avoid an
// adversarial upstream flooding logs.
func (c *Client) httpError(op string, resp *http.Response) error {
	const maxBody = 4 * 1024
	b, _ := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	return fmt.Errorf("pinata: %s: status=%d auth=*** body=%q", op, resp.StatusCode, strings.TrimSpace(string(b)))
}
