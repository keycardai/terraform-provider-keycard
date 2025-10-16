package client

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// NewTokenSource creates an OAuth2 token source for Keycard API authentication.
// The token source automatically handles token caching and refresh.
func NewTokenSource(ctx context.Context, clientID, clientSecret, organizationID, endpoint string) oauth2.TokenSource {
	tokenURL := fmt.Sprintf("%s/organizations/%s/service-account-token", endpoint, organizationID)

	config := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     tokenURL,
	}

	return config.TokenSource(ctx)
}
