package filebase

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/isongjosiah/lbvr-med/internal/config"
)

// newClientForS3Server wires an s3.Client pointing at srv. UsePathStyle
// keeps the bucket name in the path so the httptest handler sees a
// predictable /bucket/key layout.
func newClientForS3Server(t *testing.T, srv *httptest.Server, gw *httptest.Server) *Client {
	t.Helper()
	awsCfg := aws.Config{
		Region:      "us-east-1",
		Credentials: credentials.NewStaticCredentialsProvider("AKIA-test", "SECRET-test", ""),
		HTTPClient:  srv.Client(),
	}
	s3c := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(srv.URL)
		o.UsePathStyle = true
	})

	gwURL := ""
	var gwClient *http.Client
	if gw != nil {
		gwURL = gw.URL
		gwClient = gw.Client()
	}

	c, err := New(
		&config.Config{
			FilebaseAccessKey: "AKIA-test",
			FilebaseSecretKey: "SECRET-test",
			FilebaseBucket:    "lbvr-test",
		},
		WithS3Client(s3c),
		WithEndpoint(srv.URL),
		WithGatewayBase(gwURL),
		WithHTTPClient(gwClient),
	)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestNew_RejectsMissingFields(t *testing.T) {
	cases := []*config.Config{
		nil,
		{},
		{FilebaseAccessKey: "a"},
		{FilebaseAccessKey: "a", FilebaseSecretKey: "s"},
	}
	for i, cfg := range cases {
		if _, err := New(cfg); err == nil {
			t.Fatalf("case %d: expected error", i)
		}
	}
}

func TestNew_ComplainsLegiblyAndSetsDefaults(t *testing.T) {
	c, err := New(&config.Config{
		FilebaseAccessKey: "a", FilebaseSecretKey: "s", FilebaseBucket: "bkt",
	})
	if err != nil {
		t.Fatal(err)
	}
	if c.Name() != "filebase" || c.TierClass() != 1 {
		t.Fatal("Name/TierClass wrong")
	}
	if !strings.Contains(c.gatewayBase, "bkt.s3.filebase.com") {
		t.Fatalf("gatewayBase = %s", c.gatewayBase)
	}
}

func TestPut_HappyPath_ReturnsCIDFromHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/lbvr-test/") {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("x-amz-meta-cid", "QmFilebaseCid")
		w.Header().Set("ETag", `"etag"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newClientForS3Server(t, srv, nil)
	cid, err := c.Put(context.Background(), []byte("payload-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	if cid != "QmFilebaseCid" {
		t.Fatalf("cid = %q", cid)
	}
}

func TestPut_MissingCIDHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("ETag", `"etag"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newClientForS3Server(t, srv, nil)
	if _, err := c.Put(context.Background(), []byte("x")); err == nil {
		t.Fatal("expected error when x-amz-meta-cid missing")
	}
}

func TestPut_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newClientForS3Server(t, srv, nil)
	_, err := c.Put(context.Background(), []byte("x"))
	if err == nil {
		t.Fatal("expected error")
	}
	// SDK error text must not leak secret key material.
	if strings.Contains(err.Error(), "SECRET-test") {
		t.Fatalf("error leaked secret: %v", err)
	}
}

func TestPut_EmptyAndOversize(t *testing.T) {
	c, err := New(&config.Config{
		FilebaseAccessKey: "a", FilebaseSecretKey: "s", FilebaseBucket: "b",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Put(context.Background(), nil); err == nil {
		t.Fatal("expected error for empty data")
	}
	big := make([]byte, MaxUploadBytes+1)
	if _, err := c.Put(context.Background(), big); err == nil {
		t.Fatal("expected error for oversize data")
	}
}

func TestGet_HappyPath_ViaCIDGateway(t *testing.T) {
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ipfs/QmX" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte("hello-filebase"))
	}))
	defer gw.Close()

	// S3 server is unused on this path but required by constructor.
	s3srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer s3srv.Close()

	c := newClientForS3Server(t, s3srv, gw)
	data, err := c.Get(context.Background(), "QmX")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello-filebase" {
		t.Fatalf("data = %q", data)
	}
}

func TestGet_404(t *testing.T) {
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer gw.Close()
	s3srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer s3srv.Close()

	c := newClientForS3Server(t, s3srv, gw)
	if _, err := c.Get(context.Background(), "QmBad"); err == nil {
		t.Fatal("expected error")
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
	s3srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer s3srv.Close()

	c := newClientForS3Server(t, s3srv, gw)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := c.Get(ctx, "QmSlow"); err == nil {
		t.Fatal("expected timeout")
	}
}

func TestStat_HappyPath_HEADGateway(t *testing.T) {
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("method = %s", r.Method)
		}
		w.Header().Set("Content-Length", "1234")
		w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
	}))
	defer gw.Close()
	s3srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer s3srv.Close()

	c := newClientForS3Server(t, s3srv, gw)
	st, err := c.Stat(context.Background(), "QmStat")
	if err != nil {
		t.Fatal(err)
	}
	if st.CID != "QmStat" || st.SizeBytes != 1234 {
		t.Fatalf("stat = %+v", st)
	}
	if st.StoredAt.IsZero() {
		t.Fatal("StoredAt unparsed")
	}
}

func TestDelete_HappyPath(t *testing.T) {
	var sawDelete bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			sawDelete = true
			if !strings.Contains(r.URL.Path, "/QmBye") {
				t.Errorf("path = %s", r.URL.Path)
			}
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newClientForS3Server(t, srv, nil)
	if err := c.Delete(context.Background(), "QmBye"); err != nil {
		t.Fatal(err)
	}
	if !sawDelete {
		t.Fatal("handler did not see DELETE")
	}
}

func TestDeleteByKey_HappyPath(t *testing.T) {
	var key string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			key = r.URL.Path
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newClientForS3Server(t, srv, nil)
	if err := c.DeleteByKey(context.Background(), "abcdef123"); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(key, "/abcdef123") {
		t.Fatalf("unexpected key path %s", key)
	}
}

func TestDelete_EmptyInputs(t *testing.T) {
	c, _ := New(&config.Config{
		FilebaseAccessKey: "a", FilebaseSecretKey: "s", FilebaseBucket: "b",
	})
	if err := c.Delete(context.Background(), ""); err == nil {
		t.Fatal("expected empty cid error")
	}
	if err := c.DeleteByKey(context.Background(), ""); err == nil {
		t.Fatal("expected empty key error")
	}
}

func TestKeyFor_Deterministic(t *testing.T) {
	k1 := KeyFor([]byte("same bytes"))
	k2 := KeyFor([]byte("same bytes"))
	k3 := KeyFor([]byte("different"))
	if k1 != k2 {
		t.Fatal("expected deterministic key")
	}
	if k1 == k3 {
		t.Fatal("expected different bytes to produce different keys")
	}
	if len(k1) != 64 {
		t.Fatalf("expected 64-char hex, got %d", len(k1))
	}
}
