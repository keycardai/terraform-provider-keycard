package client

import (
	"context"
	"errors"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// mockTokenSource is a mock implementation of oauth2.TokenSource for testing.
type mockTokenSource struct {
	token *oauth2.Token
	err   error
}

// Token returns the configured token or error.
func (m *mockTokenSource) Token() (*oauth2.Token, error) {
	return m.token, m.err
}

// createTestJWT creates a JWT token string with the given claims for testing.
func createTestJWT(claims jwt.MapClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Sign with a dummy key (doesn't matter since we use ParseUnverified in the code)
	tokenString, _ := token.SignedString([]byte("test-secret"))
	return tokenString
}

func TestKeycardClient_GetOrganizationID(t *testing.T) {
	tests := []struct {
		name        string
		tokenSource *mockTokenSource
		wantOrgID   string
		wantErr     string
	}{
		{
			name: "success - valid token with organization_id claim",
			tokenSource: &mockTokenSource{
				token: &oauth2.Token{
					AccessToken: createTestJWT(jwt.MapClaims{
						"https://api.keycard.sh/organization_id": "org-12345",
						"sub":                                    "user@example.com",
						"exp":                                    1234567890,
					}),
				},
				err: nil,
			},
			wantOrgID: "org-12345",
			wantErr:   "",
		},
		{
			name: "success - organization_id with special characters",
			tokenSource: &mockTokenSource{
				token: &oauth2.Token{
					AccessToken: createTestJWT(jwt.MapClaims{
						"https://api.keycard.sh/organization_id": "org-abc-123-xyz",
					}),
				},
				err: nil,
			},
			wantOrgID: "org-abc-123-xyz",
			wantErr:   "",
		},
		{
			name: "error - token retrieval fails",
			tokenSource: &mockTokenSource{
				token: nil,
				err:   errors.New("oauth2 token expired"),
			},
			wantOrgID: "",
			wantErr:   "failed to get access token: oauth2 token expired",
		},
		{
			name: "error - invalid JWT token",
			tokenSource: &mockTokenSource{
				token: &oauth2.Token{
					AccessToken: "not-a-valid-jwt-token",
				},
				err: nil,
			},
			wantOrgID: "",
			wantErr:   "failed to parse JWT token:",
		},
		{
			name: "error - empty JWT token",
			tokenSource: &mockTokenSource{
				token: &oauth2.Token{
					AccessToken: "",
				},
				err: nil,
			},
			wantOrgID: "",
			wantErr:   "failed to parse JWT token:",
		},
		{
			name: "error - missing organization_id claim",
			tokenSource: &mockTokenSource{
				token: &oauth2.Token{
					AccessToken: createTestJWT(jwt.MapClaims{
						"sub": "user@example.com",
						"exp": 1234567890,
					}),
				},
				err: nil,
			},
			wantOrgID: "",
			wantErr:   "organization_id claim not found or not a string in JWT token",
		},
		{
			name: "error - organization_id claim is not a string (number)",
			tokenSource: &mockTokenSource{
				token: &oauth2.Token{
					AccessToken: createTestJWT(jwt.MapClaims{
						"https://api.keycard.sh/organization_id": 12345,
					}),
				},
				err: nil,
			},
			wantOrgID: "",
			wantErr:   "organization_id claim not found or not a string in JWT token",
		},
		{
			name: "error - organization_id claim is not a string (object)",
			tokenSource: &mockTokenSource{
				token: &oauth2.Token{
					AccessToken: createTestJWT(jwt.MapClaims{
						"https://api.keycard.sh/organization_id": map[string]interface{}{"id": "org-12345"},
					}),
				},
				err: nil,
			},
			wantOrgID: "",
			wantErr:   "organization_id claim not found or not a string in JWT token",
		},
		{
			name: "error - organization_id claim is not a string (boolean)",
			tokenSource: &mockTokenSource{
				token: &oauth2.Token{
					AccessToken: createTestJWT(jwt.MapClaims{
						"https://api.keycard.sh/organization_id": true,
					}),
				},
				err: nil,
			},
			wantOrgID: "",
			wantErr:   "organization_id claim not found or not a string in JWT token",
		},
		{
			name: "error - organization_id claim is empty string",
			tokenSource: &mockTokenSource{
				token: &oauth2.Token{
					AccessToken: createTestJWT(jwt.MapClaims{
						"https://api.keycard.sh/organization_id": "",
					}),
				},
				err: nil,
			},
			wantOrgID: "",
			wantErr:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create KeycardClient with mock token source
			// We can pass nil for ClientWithResponsesInterface since GetOrganizationID doesn't use it
			testClient := &KeycardClient{
				ClientWithResponsesInterface: nil,
				tokenSource:                  tt.tokenSource,
			}

			ctx := context.Background()
			gotOrgID, gotErr := testClient.GetOrganizationID(ctx)

			if tt.wantErr != "" {
				require.Error(t, gotErr)
				require.ErrorContains(t, gotErr, tt.wantErr)
			} else {
				require.NoError(t, gotErr)
			}

			require.Equal(t, tt.wantOrgID, gotOrgID)
		})
	}
}
