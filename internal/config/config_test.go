package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_FromDotEnvFile(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	body := "PINATA_JWT=jwt-from-file\nFILEBASE_BUCKET=from-file-bucket\n"
	if err := os.WriteFile(envPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	// Prior process env should be overridden by the file (godotenv.Overload).
	t.Setenv("PINATA_JWT", "prior-env-value")
	t.Setenv("FILEBASE_BUCKET", "prior-env-bucket")

	cfg, err := Load(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PinataJWT != "jwt-from-file" {
		t.Fatalf("PinataJWT = %q, want %q", cfg.PinataJWT, "jwt-from-file")
	}
	if cfg.FilebaseBucket != "from-file-bucket" {
		t.Fatalf("FilebaseBucket = %q, want %q", cfg.FilebaseBucket, "from-file-bucket")
	}
}

func TestLoad_MissingFileFallsBackToEnv(t *testing.T) {
	// No .env: process env should win.
	t.Setenv("PINATA_JWT", "env-only-jwt")
	t.Setenv("PINATA_GATEWAY", "https://gw.example")
	t.Setenv("FILEBASE_ACCESS_KEY", "AKIA-test")

	cfg, err := Load(filepath.Join(t.TempDir(), "missing.env"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PinataJWT != "env-only-jwt" {
		t.Fatalf("PinataJWT = %q", cfg.PinataJWT)
	}
	if cfg.PinataGateway != "https://gw.example" {
		t.Fatalf("PinataGateway = %q", cfg.PinataGateway)
	}
	if cfg.FilebaseAccessKey != "AKIA-test" {
		t.Fatalf("FilebaseAccessKey = %q", cfg.FilebaseAccessKey)
	}
}

func TestLoad_EmptyPathDefaultsToDotEnv(t *testing.T) {
	// cd into a temp dir so the implicit ".env" lookup is isolated.
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".env", []byte("PINATA_JWT=implicit\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PinataJWT != "implicit" {
		t.Fatalf("PinataJWT = %q, want %q", cfg.PinataJWT, "implicit")
	}
}

func TestLoad_UnreadablePathReturnsError(t *testing.T) {
	dir := t.TempDir()
	badPath := filepath.Join(dir, "nested")
	if err := os.Mkdir(badPath, 0o755); err != nil {
		t.Fatal(err)
	}
	// Passing a directory to godotenv.Overload fails at Open time.
	if _, err := Load(badPath); err == nil {
		t.Fatal("expected error when path is a directory, got nil")
	}
}

func TestLoad_LeavesUnsetFieldsEmpty(t *testing.T) {
	// No .env, clear every env var we care about.
	for _, k := range []string{
		"PINATA_JWT", "PINATA_GATEWAY",
		"FILEBASE_ACCESS_KEY", "FILEBASE_SECRET_KEY", "FILEBASE_BUCKET",
		"IRYS_NODE_URL", "IRYS_PRIVATE_KEY",
	} {
		t.Setenv(k, "")
	}
	cfg, err := Load(filepath.Join(t.TempDir(), "absent.env"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PinataJWT != "" || cfg.FilebaseBucket != "" || cfg.IrysNodeURL != "" {
		t.Fatalf("expected empty strings, got %+v", cfg)
	}
}
