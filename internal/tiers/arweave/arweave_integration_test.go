//go:build integration

package arweave

import (
	"context"
	"testing"
	"time"

	"github.com/isongjosiah/lbvr-med/internal/config"
)

// TestIntegration_GetAndStat_RealIrys is a minimal smoke test against the
// live Irys gateway/node. It does NOT exercise Put — the stub cannot
// produce a real tx id. Once D6 wires in ANS-104 signing this test
// should be extended to a full round-trip.
func TestIntegration_GetAndStat_RealIrys(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.IrysNodeURL == "" {
		t.Skip("IRYS_NODE_URL not set; skipping")
	}

	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Known-404 probe: confirms the wire path is reachable.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := c.Stat(ctx, "ar-this-will-not-exist"); err == nil {
		t.Fatal("expected 404 for bogus id")
	}
}
