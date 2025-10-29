package client

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// NewTokenSource creates an OAuth2 token source for Keycard API authentication.
// The token source automatically handles token caching and refresh, with retry
// logic for 429 (rate limit) and 5xx errors on token fetch operations.
func NewTokenSource(clientID, clientSecret, endpoint string) oauth2.TokenSource {
	tokenURL := fmt.Sprintf("%s/service-account-token", endpoint)

	// Create a retryable HTTP client that will handle 429 and 5xx errors
	// when fetching and refreshing tokens
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.RetryWaitMin = 2 * time.Second
	retryClient.Logger = nil

	// Convert to standard HTTP client
	standardHTTPClient := retryClient.StandardClient()
	// TODO: determine what this should actually be
	standardHTTPClient.Timeout = 5 * time.Second

	// Add the HTTP client to the context for OAuth2 to use
	// This ensures token fetches benefit from retry logic
	ctxWithClient := context.WithValue(context.Background(), oauth2.HTTPClient, standardHTTPClient)

	config := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     tokenURL,
		AuthStyle:    oauth2.AuthStyleInParams,
	}

	return config.TokenSource(ctxWithClient)
}
