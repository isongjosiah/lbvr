package arweave

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/isongjosiah/lbvr-med/internal/config"
)

func mustClient(t *testing.T, node, gw *httptest.Server) *Client {
	t.Helper()
	c, err := New(
		&config.Config{IrysNodeURL: node.URL},
		WithGateway(gw.URL),
		WithHTTPClient(gw.Client()),
	)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestNew_DefaultsAndNilConfig(t *testing.T) {
	if _, err := New(nil); err == nil {
		t.Fatal("expected error for nil config")
	}
	c, err := New(&config.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if c.nodeURL != defaultNodeURL {
		t.Fatalf("nodeURL = %s", c.nodeURL)
	}
	if c.gateway != defaultGateway {
		t.Fatalf("gateway = %s", c.gateway)
	}
	if c.Name() != "arweave" || c.TierClass() != 2 {
		t.Fatal("Name/TierClass wrong")
	}
}

func TestPut_StubIsDeterministic(t *testing.T) {
	c, err := New(&config.Config{})
	if err != nil {
		t.Fatal(err)
	}
	id1, err := c.Put(context.Background(), []byte("alpha"))
	if err != nil {
		t.Fatal(err)
	}
	id2, err := c.Put(context.Background(), []byte("alpha"))
	if err != nil {
		t.Fatal(err)
	}
	if id1 != id2 {
		t.Fatalf("stub Put should be deterministic: %s vs %s", id1, id2)
	}
	if !strings.HasPrefix(id1, "ar-") {
		t.Fatalf("stub id should be prefixed with 'ar-': %s", id1)
	}
	id3, err := c.Put(context.Background(), []byte("beta"))
	if err != nil {
		t.Fatal(err)
	}
	if id3 == id1 {
		t.Fatal("different bytes produced same stub id")
	}
}

func TestPut_RejectsEmpty(t *testing.T) {
	c, _ := New(&config.Config{})
	if _, err := c.Put(context.Background(), nil); err == nil {
		t.Fatal("expected error for empty payload")
	}
}

func TestPut_RespectsContext(t *testing.T) {
	c, _ := New(&config.Config{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Put(ctx, []byte("x")); err == nil {
		t.Fatal("expected context error")
	}
}

func TestGet_HappyPath(t *testing.T) {
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ar-abcd" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte("from-arweave"))
	}))
	defer gw.Close()
	node := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer node.Close()

	c := mustClient(t, node, gw)
	data, err := c.Get(context.Background(), "ar-abcd")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "from-arweave" {
		t.Fatalf("data = %q", data)
	}
}

func TestGet_404(t *testing.T) {
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer gw.Close()
	node := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer node.Close()

	c := mustClient(t, node, gw)
	_, err := c.Get(context.Background(), "ar-missing")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want 404 error, got %v", err)
	}
}

func TestGet_EmptyCID(t *testing.T) {
	c, _ := New(&config.Config{})
	if _, err := c.Get(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty cid")
	}
}

func TestGet_Timeout(t *testing.T) {
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(500 * time.Millisecond):
		}
	}))
	defer gw.Close()
	node := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer node.Close()

	c := mustClient(t, node, gw)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := c.Get(ctx, "ar-slow"); err == nil {
		t.Fatal("expected timeout")
	}
}

func TestStat_HappyPath(t *testing.T) {
	node := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tx/ar-abcd/status" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":         "ar-abcd",
			"size":       4242,
			"receivedAt": "2026-04-25T12:00:00Z",
		})
	}))
	defer node.Close()
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer gw.Close()

	// Stat uses c.httpClient (shared with Get); point it at gw's client — but
	// node.URL is what Stat hits. Use node's client so TLS config matches.
	c, err := New(
		&config.Config{IrysNodeURL: node.URL},
		WithGateway(gw.URL),
		WithHTTPClient(node.Client()),
	)
	if err != nil {
		t.Fatal(err)
	}
	st, err := c.Stat(context.Background(), "ar-abcd")
	if err != nil {
		t.Fatal(err)
	}
	if st.CID != "ar-abcd" || st.SizeBytes != 4242 {
		t.Fatalf("stat = %+v", st)
	}
	if st.StoredAt.IsZero() {
		t.Fatal("StoredAt unparsed")
	}
}

func TestStat_404(t *testing.T) {
	node := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer node.Close()
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer gw.Close()

	c, err := New(
		&config.Config{IrysNodeURL: node.URL},
		WithGateway(gw.URL),
		WithHTTPClient(node.Client()),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Stat(context.Background(), "ar-missing"); err == nil {
		t.Fatal("expected error")
	}
}

func TestStat_MalformedJSON(t *testing.T) {
	node := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id": "ar-x", "size":`))
	}))
	defer node.Close()
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer gw.Close()

	c, err := New(
		&config.Config{IrysNodeURL: node.URL},
		WithGateway(gw.URL),
		WithHTTPClient(node.Client()),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Stat(context.Background(), "ar-x"); err == nil {
		t.Fatal("expected decode error")
	}
}

func TestDelete_AlwaysNotImplemented(t *testing.T) {
	c, _ := New(&config.Config{})
	err := c.Delete(context.Background(), "ar-x")
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("want ErrNotImplemented, got %v", err)
	}
}
