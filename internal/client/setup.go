package client

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
)

// Config holds the configuration parameters needed to create a Keycard API client.
type Config struct {
	ClientID       string
	ClientSecret   string
	OrganizationID string
	Endpoint       string
}

// NewAPIClient creates a fully configured Keycard API client with:
// - OAuth2 authentication with automatic token refresh
// - Retry logic for 429 (rate limit) and 5xx errors on token operations
// - Request/response logging
//
// This is the primary function that should be called from the provider
// to set up the API client.
func NewAPIClient(ctx context.Context, config Config) (*ClientWithResponses, error) {
	// Create OAuth2 token source with built-in retry support for token operations
	tokenSource := NewTokenSource(config.ClientID, config.ClientSecret, config.OrganizationID, config.Endpoint)

	// Create OAuth2-authenticated HTTP client
	// This client will automatically add Bearer tokens to all requests
	oauthClient := oauth2.NewClient(ctx, tokenSource)

	// Wrap with our logging client to capture request/response details
	loggingClient := NewLoggingHTTPClient(oauthClient)

	// Create the OpenAPI-generated API client
	apiClient, err := NewClientWithResponses(config.Endpoint, WithHTTPClient(loggingClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	return apiClient, nil
}
