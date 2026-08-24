package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// writeTestFile writes content to a file under a temp dir and returns its path.
func writeTestFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// TestLoad_InvalidJSONReturnsError reproduces the reported bug: an invalid
// config file must surface an error instead of silently falling back to the
// default configuration.
func TestLoad_InvalidJSONReturnsError(t *testing.T) {
	invalidPath := writeTestFile(t, "invalid.json", "this is not valid json")

	cfg, err := Load(invalidPath)
	if err == nil {
		t.Fatalf("expected an error for an invalid config file, got nil; cfg=%+v", cfg)
	}
	if !strings.Contains(err.Error(), "failed to load config file") {
		t.Fatalf("error should mention config file load failure, got: %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected nil cfg on error, got non-nil")
	}
}

// TestLoad_ValidFile applies the file's values.
func TestLoad_ValidFile(t *testing.T) {
	// time.Duration unmarshals from a JSON number of nanoseconds.
	const (
		fiveSeconds       = 5_000_000_000
		tenSeconds        = 10_000_000_000
		twoKB     int64   = 2048
	)
	validJSON := `{"server":{"host":"127.0.0.1","port":9090,"read_timeout":` +
		itoaInt(fiveSeconds) + `,"write_timeout":` + itoaInt(fiveSeconds) +
		`,"idle_timeout":` + itoaInt(tenSeconds) + `,"max_request_body_size":` +
		itoaInt64(twoKB) + `,"shutdown_timeout":` + itoaInt(fiveSeconds) + `}}`
	validPath := writeTestFile(t, "valid.json", validJSON)

	cfg, err := Load(validPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server.Host != "127.0.0.1" {
		t.Fatalf("expected host from file, got %q", cfg.Server.Host)
	}
	if cfg.Server.Port != 9090 {
		t.Fatalf("expected port 9090, got %d", cfg.Server.Port)
	}
}

func itoaInt(n int) string   { return strconv.Itoa(n) }
func itoaInt64(n int64) string { return strconv.FormatInt(n, 10) }

// TestLoad_MissingFileFallsBackToDefault with an empty path should yield
// the default config without error.
func TestLoad_MissingFileFallsBackToDefault(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	def := DefaultConfig()
	if cfg.Server.Port != def.Server.Port {
		t.Fatalf("expected default port %d, got %d", def.Server.Port, cfg.Server.Port)
	}
}
