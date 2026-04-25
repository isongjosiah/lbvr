//go:build integration

package filebase

import (
	"bytes"
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/isongjosiah/lbvr-med/internal/config"
)

func TestIntegration_PutGetStatDelete(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.FilebaseAccessKey == "" || cfg.FilebaseSecretKey == "" || cfg.FilebaseBucket == "" {
		t.Skip("Filebase credentials not set; skipping real-API integration test")
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
	t.Logf("filebase put cid=%s", cid)

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

	// Use DeleteByKey so we delete by the deterministic S3 key we uploaded under.
	if err := c.DeleteByKey(ctx, KeyFor(payload)); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
