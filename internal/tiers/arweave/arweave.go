// Package arweave implements the cold-tier (CLAUDE.md §4.5) tiers.Client
// against an Irys node's REST surface. This is the WEAKEST of the three
// tier clients as of D4 — the Irys upload protocol requires a signed
// data-item bundle (Arweave ANS-104) and the Irys SDK ecosystem is
// JS-first. A full Go implementation requires either (a) goar
// (github.com/everFinance/goar), which pulls in go-ethereum and
// ~50 MB of transitive crypto deps, or (b) a hand-rolled ANS-104
// signer.
//
// Conference scope decision (CLAUDE.md §3.1 + the D4 brief): ship a
// stub that implements tiers.Client faithfully at the API level so the
// D6 ingest CLI and D8 gateway can compile and exercise the fast path
// against the hot + warm tiers. The cold-tier path is exercised in E9
// via Toxiproxy-injected stubs of the expected Irys responses.
//
// Put: synthesises a tx id as "ar-" + hex(sha256(data)). This is
// stable, dedupable, and clearly non-Arweave-native (the "ar-" prefix
// is not part of the real Arweave base64url-id format).
//
// Get: hits the configured Irys gateway (default https://gateway.irys.xyz)
// at /<tx> and returns the body. Real uploads will become retrievable
// once the goar (or hand-rolled) signer is wired in on D6+; stub
// uploads will 404 here until the backend actually holds the bytes.
//
// Stat: GET /tx/<id>/status on the Irys node URL; returns whatever
// Irys reports. Documented as "best-effort" until Put is real.
//
// Delete: returns ErrNotImplemented — Arweave is immutable, the real
// client should never call Delete on a confirmed tx. The retrieval
// gateway's PoR layer is the mechanism for dealing with unavailable
// cold-tier shards (tier migration, see CLAUDE.md §4.4).
//
// To finish this package (D6 or later), the user needs to either (a)
// add github.com/everFinance/goar and replace signAndUpload with a
// goar.Client.SendBundle call, or (b) implement ANS-104 data-item
// construction + Ed25519/secp256k1 signing directly and POST to
// <IRYS_NODE_URL>/tx/<token>. See https://docs.irys.xyz/developer-docs/http-api.
package arweave

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/isongjosiah/lbvr-med/internal/config"
	"github.com/isongjosiah/lbvr-med/internal/tiers"
)

// uploadStubEnabled toggles Put's stub: true → synthesise a stub id,
// false → return ErrNotImplemented. Flip to false on D6+ once signing works.
const uploadStubEnabled = true

const (
	defaultNodeURL = "https://node1.irys.xyz"
	defaultGateway = "https://gateway.irys.xyz"
	defaultTimeout = 60 * time.Second
	maxGetBytes    = 128 * 1024 * 1024
	stubTxPrefix   = "ar-"
)

// ErrNotImplemented signals that a method is a deliberate stub. D6+ will
// replace the Put implementation; Delete stays ErrNotImplemented forever.
var ErrNotImplemented = errors.New("arweave: not implemented (stub — see package doc)")

// Client is the Arweave / Irys cold-tier client.
type Client struct {
	httpClient *http.Client
	nodeURL    string
	gateway    string
	privKey    string // currently unused; kept so D6 wiring is source-compat
}

// Option customises a Client.
type Option func(*Client)

// WithHTTPClient overrides the default HTTP client.
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.httpClient = h } }

// WithNodeURL overrides the Irys node URL.
func WithNodeURL(u string) Option { return func(c *Client) { c.nodeURL = u } }

// WithGateway overrides the Irys gateway URL used by Get.
func WithGateway(u string) Option { return func(c *Client) { c.gateway = u } }

// New constructs an Arweave Client. The private key may be empty; the
// stub Put does not use it. A real Put (D6) MUST error on empty.
func New(cfg *config.Config, opts ...Option) (*Client, error) {
	if cfg == nil {
		return nil, errors.New("arweave: nil config")
	}
	node := cfg.IrysNodeURL
	if node == "" {
		node = defaultNodeURL
	}
	c := &Client{
		httpClient: &http.Client{Timeout: defaultTimeout},
		nodeURL:    strings.TrimRight(node, "/"),
		gateway:    defaultGateway,
		privKey:    cfg.IrysPrivateKey,
	}
	for _, o := range opts {
		o(c)
	}
	return c, nil
}

// Name implements tiers.Client.
func (c *Client) Name() string { return "arweave" }

// TierClass implements tiers.Client.
func (c *Client) TierClass() uint8 { return tiers.TierCold }

// Put synthesises a deterministic stub tx id (see package doc). The
// bytes are NOT actually uploaded in the current stub. TODO(D6+):
// replace with a real ANS-104-signed bundle upload.
func (c *Client) Put(ctx context.Context, data []byte) (string, error) {
	if !uploadStubEnabled {
		return "", ErrNotImplemented
	}
	if len(data) == 0 {
		return "", errors.New("arweave: empty payload")
	}
	if int64(len(data)) > maxGetBytes {
		return "", fmt.Errorf("arweave: payload %d bytes exceeds cap %d", len(data), maxGetBytes)
	}
	// Honour ctx so callers with very-short deadlines still bail out.
	if err := ctx.Err(); err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return stubTxPrefix + hex.EncodeToString(h[:]), nil
}

// Get fetches the bytes for a tx id from the Irys gateway. Returns a
// clean 404-shaped error for unknown ids so the retrieval gateway can
// distinguish "not yet settled" from transport failures.
func (c *Client) Get(ctx context.Context, cid string) ([]byte, error) {
	if cid == "" {
		return nil, errors.New("arweave: empty cid")
	}
	u := c.gateway + "/" + cid
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("arweave: req: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("arweave: http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("arweave: tx %s not found (status=404)", cid)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("arweave: get status=%d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxGetBytes+1))
}

// irysStatus is the subset of /tx/<id>/status we consume.
// Deadline is unix seconds; currently unused but documented for D6+.
// ReceivedAt is RFC3339.
type irysStatus struct {
	ID         string `json:"id"`
	Size       int64  `json:"size"`
	Deadline   int64  `json:"deadline"`
	ReceivedAt string `json:"receivedAt"`
}

// Stat queries the Irys node for tx status. Marked best-effort in the
// package doc until Put is real.
func (c *Client) Stat(ctx context.Context, cid string) (*tiers.Stat, error) {
	if cid == "" {
		return nil, errors.New("arweave: empty cid")
	}
	u := c.nodeURL + "/tx/" + cid + "/status"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("arweave: req: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("arweave: http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("arweave: tx %s not found (status=404)", cid)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("arweave: stat status=%d", resp.StatusCode)
	}
	var st irysStatus
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		return nil, fmt.Errorf("arweave: decode: %w", err)
	}
	out := &tiers.Stat{CID: cid, SizeBytes: st.Size}
	if st.ReceivedAt != "" {
		if ts, err := time.Parse(time.RFC3339, st.ReceivedAt); err == nil {
			out.StoredAt = ts
		}
	}
	return out, nil
}

// Delete is a permanent stub — Arweave is immutable by design, and
// Irys-mediated uploads inherit that property. Tier migration, not
// deletion, is how LBVR-Med evicts stale cold-tier shards.
func (c *Client) Delete(_ context.Context, _ string) error {
	return ErrNotImplemented
}
