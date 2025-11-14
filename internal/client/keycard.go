package client

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

// Config holds the configuration parameters needed to create a Keycard API client.
type Config struct {
	ClientID     string
	ClientSecret string
	Endpoint     string
}

type KeycardClient struct {
	ClientWithResponsesInterface

	tokenSource oauth2.TokenSource
}

// NewAPIClient creates a fully configured Keycard API client with:
// - OAuth2 authentication with automatic token refresh
// - Retry logic for 429 (rate limit) and 5xx errors on token operations
// - Request/response logging
//
// This is the primary function that should be called from the provider
// to set up the API client.
func NewAPIClient(ctx context.Context, config Config) (*KeycardClient, error) {
	// Create OAuth2 token source with built-in retry support for token operations
	tokenSource := NewTokenSource(config.ClientID, config.ClientSecret, config.Endpoint)

	// Create OAuth2-authenticated HTTP client
	// This client will automatically add Bearer tokens to all requests
	oauthClient := oauth2.NewClient(ctx, tokenSource)

	// 5 second timeout for API operations - balances responsiveness with reliability.
	// Retry logic at the transport level handles transient failures and rate limits.
	oauthClient.Timeout = 5 * time.Second

	// Wrap with our logging client to capture request/response details
	loggingClient := NewLoggingHTTPClient(oauthClient)

	// Create the OpenAPI-generated API client
	apiClient, err := NewClientWithResponses(config.Endpoint, WithHTTPClient(loggingClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	return &KeycardClient{apiClient, tokenSource}, nil
}

// GetOrganizationID extracts the organization_id claim from the JWT access token.
// Returns an error if the token cannot be retrieved or decoded.
func (c *KeycardClient) GetOrganizationID(ctx context.Context) (string, error) {
	// Get the current access token
	token, err := c.tokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	// Parse the JWT without verification (we trust our own OAuth2 provider)
	// The token is already validated by the OAuth2 flow
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	claims := jwt.MapClaims{}

	_, _, err = parser.ParseUnverified(token.AccessToken, claims)
	if err != nil {
		return "", fmt.Errorf("failed to parse JWT token: %w", err)
	}

	// Extract organization_id from claims
	orgID, ok := claims["https://api.keycard.sh/organization_id"].(string)
	if !ok {
		return "", fmt.Errorf("organization_id claim not found or not a string in JWT token")
	}

	return orgID, nil
}
