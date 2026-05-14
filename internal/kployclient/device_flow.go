package kployclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
)

// GitHub OAuth Device Flow endpoints. These are fixed by GitHub —
// the only piece that varies per kploy environment is the client ID,
// which we fetch from the kploy server.
const (
	githubDeviceAuthURL = "https://github.com/login/device/code"
	githubTokenURL      = "https://github.com/login/oauth/access_token"
)

// DeviceFlowOAuthConfig fetches the GitHub App's client ID from
// kploy and returns a populated oauth2.Config ready for DeviceAuth
// or TokenSource. The Scopes match what the kploy GitHub App
// requests on the web side.
func DeviceFlowOAuthConfig(ctx context.Context, server string) (*oauth2.Config, error) {
	c, err := NewUnauthenticated(server)
	if err != nil {
		return nil, err
	}
	resp, err := c.GetDeviceFlowConfigWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch device-flow config: %w", err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, fmt.Errorf("device-flow config request returned %d", resp.StatusCode())
	}
	if resp.JSON200.GithubClientID == "" {
		return nil, errors.New("server returned empty github_client_id")
	}
	return &oauth2.Config{
		ClientID: resp.JSON200.GithubClientID,
		Endpoint: oauth2.Endpoint{
			DeviceAuthURL: githubDeviceAuthURL,
			TokenURL:      githubTokenURL,
		},
		Scopes: []string{"read:user"},
	}, nil
}

