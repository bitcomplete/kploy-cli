package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestSetTokenRoundtrip(t *testing.T) {
	expiry := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	c := &Config{}
	c.SetToken(&oauth2.Token{
		AccessToken:  "ghu_a",
		RefreshToken: "ghr_b",
		Expiry:       expiry,
	})
	got := c.Token()
	if got.AccessToken != "ghu_a" || got.RefreshToken != "ghr_b" || !got.Expiry.Equal(expiry) {
		t.Fatalf("roundtrip failed: %+v", got)
	}
}

func TestSetTokenNilClears(t *testing.T) {
	c := &Config{AccessToken: "x", RefreshToken: "y"}
	c.SetToken(nil)
	if c.Token() != nil {
		t.Fatalf("expected nil token after SetToken(nil)")
	}
}

func TestResolveServerPrecedence(t *testing.T) {
	t.Setenv(EnvServer, "")
	c := &Config{}
	if got := c.ResolveServer(); got != DefaultServer {
		t.Fatalf("default: got %q want %q", got, DefaultServer)
	}
	c.Server = "https://custom.example"
	if got := c.ResolveServer(); got != "https://custom.example" {
		t.Fatalf("configured: got %q", got)
	}
	t.Setenv(EnvServer, "https://envvar.example")
	if got := c.ResolveServer(); got != "https://envvar.example" {
		t.Fatalf("env-var override: got %q", got)
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg := &Config{Server: "https://s.example", Org: "acme", AccessToken: "ghu_x"}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Server != cfg.Server || got.Org != cfg.Org || got.AccessToken != cfg.AccessToken {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}
	path := filepath.Join(dir, ".config", "kploy", "config.yaml")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("config file perms = %o, want 0600", perm)
	}
}

func TestLoadMissingFileReturnsZero(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected non-nil zero config")
	}
	if got.AccessToken != "" || got.Server != "" {
		t.Errorf("expected zero-value config, got %+v", got)
	}
}
