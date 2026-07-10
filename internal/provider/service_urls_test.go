package provider

import (
	"testing"
)

func TestDeriveServiceURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		endpoint  string
		subdomain string
		want      string
		wantErr   bool
	}{
		{
			name:      "production identity",
			endpoint:  "https://api.keycard.ai",
			subdomain: "id",
			want:      "https://id.keycard.ai",
		},
		{
			name:      "production console",
			endpoint:  "https://api.keycard.ai",
			subdomain: "console",
			want:      "https://console.keycard.ai",
		},
		{
			name:      "staging identity",
			endpoint:  "https://api.staging.keycard.ai",
			subdomain: "id",
			want:      "https://id.staging.keycard.ai",
		},
		{
			name:      "with port",
			endpoint:  "https://api.keycard.ai:8443",
			subdomain: "id",
			want:      "https://id.keycard.ai:8443",
		},
		{
			name:      "with trailing path ignored",
			endpoint:  "https://api.keycard.ai/v1",
			subdomain: "id",
			want:      "https://id.keycard.ai",
		},
		{
			name:      "no api prefix",
			endpoint:  "https://localhost:8080",
			subdomain: "id",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := deriveServiceURL(tt.endpoint, tt.subdomain)
			if (err != nil) != tt.wantErr {
				t.Fatalf("deriveServiceURL() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("deriveServiceURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
