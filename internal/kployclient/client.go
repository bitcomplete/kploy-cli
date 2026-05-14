// Package kployclient constructs an authenticated kploy API client
// from CLI config. The client uses oauth2.NewClient so the GitHub
// token refreshes transparently. After every successful refresh we
// persist the new token back to the config file via the
// persistingTokenSource wrapper.
package kployclient

import (
	"context"
	"errors"
	"fmt"

	kployapi "github.com/bitcomplete/kploy-cli/client"
	"github.com/bitcomplete/kploy-cli/internal/config"
	"golang.org/x/oauth2"
)

// New builds a kploy API client with the user's stored token wired
// in. Returns ErrNotLoggedIn if no token is configured.
func New(ctx context.Context, cfg *config.Config) (*kployapi.ClientWithResponses, error) {
	tok := cfg.Token()
	if tok == nil {
		return nil, ErrNotLoggedIn
	}
	oauthCfg, err := DeviceFlowOAuthConfig(ctx, cfg.ResolveServer())
	if err != nil {
		return nil, err
	}
	base := oauthCfg.TokenSource(ctx, tok)
	persisting := &persistingTokenSource{base: base, cfg: cfg}
	httpClient := oauth2.NewClient(ctx, persisting)
	return kployapi.NewClientWithResponses(cfg.ResolveServer()+"/api/v1", kployapi.WithHTTPClient(httpClient))
}

// NewUnauthenticated builds a client without any auth header. Used
// for the device-flow-config bootstrap call and other public routes.
func NewUnauthenticated(server string) (*kployapi.ClientWithResponses, error) {
	return kployapi.NewClientWithResponses(server + "/api/v1")
}

var ErrNotLoggedIn = errors.New("not logged in; run `kploy auth login`")

type persistingTokenSource struct {
	base oauth2.TokenSource
	cfg  *config.Config
	last *oauth2.Token
}

func (p *persistingTokenSource) Token() (*oauth2.Token, error) {
	tok, err := p.base.Token()
	if err != nil {
		return nil, err
	}
	if p.last == nil || tok.AccessToken != p.last.AccessToken {
		p.cfg.SetToken(tok)
		if err := config.Save(p.cfg); err != nil {
			return nil, fmt.Errorf("persist refreshed token: %w", err)
		}
		p.last = tok
	}
	return tok, nil
}
