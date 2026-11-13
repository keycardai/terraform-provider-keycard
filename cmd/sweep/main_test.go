package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

func TestSweepable(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		zone client.Zone
		want bool
	}{
		{
			name: "old test zone",
			zone: client.Zone{Name: "tftest-zone-123", CreatedAt: now.Add(-2 * time.Hour)},
			want: true,
		},
		{
			name: "old test zone from the generic prefix",
			zone: client.Zone{Name: "tftest-123", CreatedAt: now.Add(-2 * time.Hour)},
			want: true,
		},
		{
			name: "test zone exactly at the minimum age",
			zone: client.Zone{Name: "tftest-zone-123", CreatedAt: now.Add(-minZoneAge)},
			want: true,
		},
		{
			name: "test zone a running test may still own",
			zone: client.Zone{Name: "tftest-zone-123", CreatedAt: now.Add(-10 * time.Minute)},
			want: false,
		},
		{
			name: "old zone without the test prefix",
			zone: client.Zone{Name: "Default Zone", CreatedAt: now.Add(-30 * 24 * time.Hour)},
			want: false,
		},
		{
			name: "name that starts with tftest but not the test prefix",
			zone: client.Zone{Name: "tftesting-prod", CreatedAt: now.Add(-2 * time.Hour)},
			want: false,
		},
		{
			name: "prefix appears later in the name",
			zone: client.Zone{Name: "prod-tftest-1", CreatedAt: now.Add(-2 * time.Hour)},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sweepable(tt.zone, now); got != tt.want {
				t.Errorf("sweepable(%q, created %s ago) = %v, want %v", tt.zone.Name, now.Sub(tt.zone.CreatedAt), got, tt.want)
			}
		})
	}
}

// TestRunRejectsBadArguments checks that bad arguments stop the command before
// it reads credentials, so a typo such as -dry-run=yes can never fall through
// to a real sweep.
func TestRunRejectsBadArguments(t *testing.T) {
	for _, envVar := range []string{"KEYCARD_CLIENT_ID", "KEYCARD_CLIENT_SECRET", "KEYCARD_ENDPOINT"} {
		t.Setenv(envVar, "")
	}

	tests := []struct {
		args    []string
		wantErr string
	}{
		{args: []string{"-dry-run=yes"}, wantErr: "invalid boolean value"},
		{args: []string{"-dryrun"}, wantErr: "flag provided but not defined"},
		{args: []string{"now"}, wantErr: "unexpected arguments: now"},
		// Valid flags get as far as the credentials check.
		{args: []string{"-dry-run"}, wantErr: "KEYCARD_CLIENT_ID must be set"},
		{args: nil, wantErr: "KEYCARD_CLIENT_ID must be set"},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			err := run(context.Background(), tt.args, io.Discard)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("run(%q) error = %v, want one containing %q", tt.args, err, tt.wantErr)
			}
		})
	}
}

// fakeResponse stands in for a generated *WithResponse type.
type fakeResponse struct{ status int }

func (r *fakeResponse) StatusCode() int { return r.status }

func TestWithRetry(t *testing.T) {
	original := initialBackoff
	initialBackoff = time.Millisecond
	t.Cleanup(func() { initialBackoff = original })

	errTimeout := errors.New("Client.Timeout exceeded")

	tests := []struct {
		name       string
		results    []func() (*fakeResponse, error)
		wantCalls  int
		wantStatus int
		wantErr    bool
	}{
		{
			name:       "success on the first try",
			results:    []func() (*fakeResponse, error){status(http.StatusNoContent)},
			wantCalls:  1,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "404 is not retried",
			results:    []func() (*fakeResponse, error){status(http.StatusNotFound)},
			wantCalls:  1,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "timeout, then the delete already happened",
			results:    []func() (*fakeResponse, error){fail(errTimeout), status(http.StatusNotFound)},
			wantCalls:  2,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "429 and 502 are retried",
			results:    []func() (*fakeResponse, error){status(http.StatusTooManyRequests), status(http.StatusBadGateway), status(http.StatusNoContent)},
			wantCalls:  3,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "gives up after the last attempt",
			results:    []func() (*fakeResponse, error){status(http.StatusServiceUnavailable), status(http.StatusServiceUnavailable), status(http.StatusServiceUnavailable), status(http.StatusServiceUnavailable), status(http.StatusNoContent)},
			wantCalls:  attempts,
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:      "returns the last transport error",
			results:   []func() (*fakeResponse, error){fail(errTimeout), fail(errTimeout), fail(errTimeout), fail(errTimeout)},
			wantCalls: attempts,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			resp, err := withRetry(context.Background(), func() (*fakeResponse, error) {
				result := tt.results[calls]
				calls++
				return result()
			})

			if calls != tt.wantCalls {
				t.Errorf("calls = %d, want %d", calls, tt.wantCalls)
			}
			if tt.wantErr {
				if err == nil {
					t.Fatalf("err = nil, want an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if resp.StatusCode() != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode(), tt.wantStatus)
			}
		})
	}
}

func status(code int) func() (*fakeResponse, error) {
	return func() (*fakeResponse, error) { return &fakeResponse{status: code}, nil }
}

func fail(err error) func() (*fakeResponse, error) {
	return func() (*fakeResponse, error) { return nil, err }
}
