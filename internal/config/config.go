// Package config loads and persists the CLI's on-disk state at
// ~/.config/kploy/config.yaml. Permissions are 0600 because the file
// contains the user's GitHub OAuth tokens.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

const (
	DefaultServer = "https://kploy.app"
	EnvServer     = "KPLOY_SERVER"
	EnvToken      = "KPLOY_TOKEN"
	EnvOrg        = "KPLOY_ORG"
)

type Config struct {
	Server       string     `yaml:"server,omitempty"`
	Org          string     `yaml:"org,omitempty"`
	AccessToken  string     `yaml:"access_token,omitempty"`
	RefreshToken string     `yaml:"refresh_token,omitempty"`
	Expiry       *time.Time `yaml:"expiry,omitempty"`
}

// ResolveServer returns the configured server URL or the
// $KPLOY_SERVER override or the production default.
func (c *Config) ResolveServer() string {
	if s := os.Getenv(EnvServer); s != "" {
		return s
	}
	if c.Server != "" {
		return c.Server
	}
	return DefaultServer
}

// ResolveOrg returns the configured org or the $KPLOY_ORG override.
// Returns "" if neither is set.
func (c *Config) ResolveOrg() string {
	if o := os.Getenv(EnvOrg); o != "" {
		return o
	}
	return c.Org
}

// Token returns the stored OAuth token. Empty if not logged in.
func (c *Config) Token() *oauth2.Token {
	if c.AccessToken == "" {
		return nil
	}
	t := &oauth2.Token{
		AccessToken:  c.AccessToken,
		RefreshToken: c.RefreshToken,
	}
	if c.Expiry != nil {
		t.Expiry = *c.Expiry
	}
	return t
}

func (c *Config) SetToken(t *oauth2.Token) {
	if t == nil {
		c.AccessToken = ""
		c.RefreshToken = ""
		c.Expiry = nil
		return
	}
	c.AccessToken = t.AccessToken
	c.RefreshToken = t.RefreshToken
	if !t.Expiry.IsZero() {
		expiry := t.Expiry
		c.Expiry = &expiry
	} else {
		c.Expiry = nil
	}
}

// Path returns the canonical config file path.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "kploy", "config.yaml"), nil
}

// Load reads the config, returning a zero Config if the file doesn't exist.
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &c, nil
}

// Save writes the config atomically with 0600 perms.
func Save(c *Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
