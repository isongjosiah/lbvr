//go:build integration

package pinata

import (
	"bytes"
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/isongjosiah/lbvr-med/internal/config"
)

// TestIntegration_PutGetStatDelete exercises the real Pinata API. Runs only
// with `go test -tags=integration` after PINATA_JWT is populated in .env.
func TestIntegration_PutGetStatDelete(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PinataJWT == "" {
		t.Skip("PINATA_JWT not set; skipping real-API integration test")
	}

	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	payload := make([]byte, 8*1024)
	if _, err := rand.Read(payload); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cid, err := c.Put(ctx, payload)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("pinata put cid=%s", cid)

	got, err := c.Get(ctx, cid)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("round-trip payload mismatch")
	}

	if _, err := c.Stat(ctx, cid); err != nil {
		t.Fatalf("stat: %v", err)
	}
	if err := c.Delete(ctx, cid); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
