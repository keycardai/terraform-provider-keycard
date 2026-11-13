package provider

import "testing"

func TestArchiveBlockedInUse(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		msgSubstring string
		want         bool
	}{
		{
			name:         "policy version currently active",
			body:         `{"status":400,"code":"bad_request","message":"cannot archive version: currently active","request_id":"abc","timestamp":"2026-08-18T11:03:49Z","details":[],"path":"/zones/z/policies/p/versions/v"}`,
			msgSubstring: "currently active",
			want:         true,
		},
		{
			name:         "policy set version currently bound",
			body:         `{"status":400,"code":"bad_request","message":"cannot archive version: currently bound"}`,
			msgSubstring: "currently bound",
			want:         true,
		},
		{
			name:         "policy set active bindings",
			body:         `{"status":409,"code":"conflict","message":"policy set has active bindings and cannot be archived"}`,
			msgSubstring: "active bindings",
			want:         true,
		},
		{
			name:         "different message",
			body:         `{"status":400,"code":"bad_request","message":"invalid version id"}`,
			msgSubstring: "currently active",
			want:         false,
		},
		{
			name:         "malformed body",
			body:         `<html>bad gateway</html>`,
			msgSubstring: "currently active",
			want:         false,
		},
		{
			name:         "empty body",
			body:         ``,
			msgSubstring: "currently active",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := archiveBlockedInUse([]byte(tt.body), tt.msgSubstring); got != tt.want {
				t.Errorf("archiveBlockedInUse(%q, %q) = %v, want %v", tt.body, tt.msgSubstring, got, tt.want)
			}
		})
	}
}
