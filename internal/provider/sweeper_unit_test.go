package provider

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

func TestSweepableZone(t *testing.T) {
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
			zone: client.Zone{Name: "tftest-zone-123", CreatedAt: now.Add(-sweepZoneMinAge)},
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
			if got := sweepableZone(tt.zone, now); got != tt.want {
				t.Errorf("sweepableZone(%q, created %s ago) = %v, want %v", tt.zone.Name, now.Sub(tt.zone.CreatedAt), got, tt.want)
			}
		})
	}
}

func TestSweepDryRun(t *testing.T) {
	tests := []struct {
		value   string
		want    bool
		wantErr bool
	}{
		{value: "", want: false},
		{value: "1", want: true},
		{value: "true", want: true},
		{value: "0", want: false},
		{value: "false", want: false},
		{value: "yes", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(sweepDryRunEnv+"="+tt.value, func(t *testing.T) {
			t.Setenv(sweepDryRunEnv, tt.value)
			got, err := sweepDryRun()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("err = nil, want an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if got != tt.want {
				t.Errorf("sweepDryRun() = %v, want %v", got, tt.want)
			}
		})
	}
}

// fakeSweepResponse stands in for a generated *WithResponse type.
type fakeSweepResponse struct{ status int }

func (r *fakeSweepResponse) StatusCode() int { return r.status }

func TestWithSweepRetry(t *testing.T) {
	original := sweepInitialBackoff
	sweepInitialBackoff = time.Millisecond
	t.Cleanup(func() { sweepInitialBackoff = original })

	errTimeout := errors.New("Client.Timeout exceeded")

	tests := []struct {
		name       string
		results    []func() (*fakeSweepResponse, error)
		wantCalls  int
		wantStatus int
		wantErr    bool
	}{
		{
			name:       "success on the first try",
			results:    []func() (*fakeSweepResponse, error){sweepStatus(http.StatusNoContent)},
			wantCalls:  1,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "404 is not retried",
			results:    []func() (*fakeSweepResponse, error){sweepStatus(http.StatusNotFound)},
			wantCalls:  1,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "timeout, then the delete already happened",
			results:    []func() (*fakeSweepResponse, error){sweepFail(errTimeout), sweepStatus(http.StatusNotFound)},
			wantCalls:  2,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "429 and 502 are retried",
			results:    []func() (*fakeSweepResponse, error){sweepStatus(http.StatusTooManyRequests), sweepStatus(http.StatusBadGateway), sweepStatus(http.StatusNoContent)},
			wantCalls:  3,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "gives up after the last attempt",
			results:    []func() (*fakeSweepResponse, error){sweepStatus(http.StatusServiceUnavailable), sweepStatus(http.StatusServiceUnavailable), sweepStatus(http.StatusServiceUnavailable), sweepStatus(http.StatusServiceUnavailable), sweepStatus(http.StatusNoContent)},
			wantCalls:  sweepAttempts,
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:      "returns the last transport error",
			results:   []func() (*fakeSweepResponse, error){sweepFail(errTimeout), sweepFail(errTimeout), sweepFail(errTimeout), sweepFail(errTimeout)},
			wantCalls: sweepAttempts,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			resp, err := withSweepRetry(context.Background(), func() (*fakeSweepResponse, error) {
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

func sweepStatus(code int) func() (*fakeSweepResponse, error) {
	return func() (*fakeSweepResponse, error) { return &fakeSweepResponse{status: code}, nil }
}

func sweepFail(err error) func() (*fakeSweepResponse, error) {
	return func() (*fakeSweepResponse, error) { return nil, err }
}
