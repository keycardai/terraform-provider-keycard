package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/oapi-codegen/nullable"
)

func TestSweepable(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		target target
		want   bool
	}{
		{
			name:   "old test zone",
			target: target{name: "tftest-zone-123", createdAt: now.Add(-2 * time.Hour)},
			want:   true,
		},
		{
			name:   "old test zone from the generic prefix",
			target: target{name: "tftest-123", createdAt: now.Add(-2 * time.Hour)},
			want:   true,
		},
		{
			name:   "test zone exactly at the minimum age",
			target: target{name: "tftest-zone-123", createdAt: now.Add(-minAge)},
			want:   true,
		},
		{
			name:   "test zone a running test may still own",
			target: target{name: "tftest-zone-123", createdAt: now.Add(-10 * time.Minute)},
			want:   false,
		},
		{
			name:   "test zone between one and two hours old",
			target: target{name: "tftest-zone-123", createdAt: now.Add(-90 * time.Minute)},
			want:   true,
		},
		{
			name:   "test zone just below the minimum age",
			target: target{name: "tftest-zone-123", createdAt: now.Add(-minAge + time.Second)},
			want:   false,
		},
		{
			name:   "old zone without the test prefix",
			target: target{name: "Default Zone", createdAt: now.Add(-30 * 24 * time.Hour)},
			want:   false,
		},
		{
			name:   "name that starts with tftest but not the test prefix",
			target: target{name: "tftesting-prod", createdAt: now.Add(-2 * time.Hour)},
			want:   false,
		},
		{
			name:   "prefix appears later in the name",
			target: target{name: "prod-tftest-1", createdAt: now.Add(-2 * time.Hour)},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sweepable(tt.target, now); got != tt.want {
				t.Errorf("sweepable(%q, created %s ago) = %v, want %v", tt.target.name, now.Sub(tt.target.createdAt), got, tt.want)
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

func TestListAllFollowsCursor(t *testing.T) {
	pages := map[string]listPage{
		"":   {reply: reply{status: http.StatusOK}, ok: true, targets: []target{{id: "a"}}, next: nullable.NewNullableWithValue("c1")},
		"c1": {reply: reply{status: http.StatusOK}, ok: true, targets: []target{{id: "b"}}, next: nullable.NewNullNullable[string]()},
	}
	col := collection{
		kind: "resource",
		list: func(_ context.Context, after *string) (listPage, error) {
			cursor := ""
			if after != nil {
				cursor = *after
			}
			return pages[cursor], nil
		},
	}

	got, err := listAll(context.Background(), col)
	if err != nil {
		t.Fatalf("listAll: %v", err)
	}
	if len(got) != 2 || got[0].id != "a" || got[1].id != "b" {
		t.Errorf("listAll = %v, want [a b]", got)
	}
}

func TestListAllFailsOnErrorStatus(t *testing.T) {
	col := collection{
		kind: "resource",
		list: func(context.Context, *string) (listPage, error) {
			return listPage{reply: reply{status: http.StatusForbidden, body: []byte("denied")}}, nil
		},
	}

	_, err := listAll(context.Background(), col)
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Errorf("listAll error = %v, want a 403 error", err)
	}
}

func TestSweepCollection(t *testing.T) {
	old := time.Now().Add(-2 * time.Hour)
	targets := []target{
		{id: "stale", name: "tftest-1", createdAt: old},
		{id: "gone", name: "tftest-2", createdAt: old},
		{id: "broken", name: "tftest-3", createdAt: old},
		{id: "new", name: "tftest-4", createdAt: time.Now()},
		{id: "prod", name: "prod", createdAt: old},
	}
	statuses := map[string]int{"stale": http.StatusNoContent, "gone": http.StatusNotFound, "broken": http.StatusConflict}

	newCollection := func(deleted *[]string) collection {
		var mu sync.Mutex
		return collection{
			kind: "application",
			list: func(context.Context, *string) (listPage, error) {
				return listPage{reply: reply{status: http.StatusOK}, ok: true, targets: targets}, nil
			},
			delete: func(_ context.Context, id string) (reply, error) {
				mu.Lock()
				defer mu.Unlock()
				*deleted = append(*deleted, id)
				return reply{status: statuses[id]}, nil
			},
		}
	}

	t.Run("dry run deletes nothing", func(t *testing.T) {
		var deleted []string
		if err := sweepCollection(context.Background(), newCollection(&deleted), true); err != nil {
			t.Fatalf("sweepCollection: %v", err)
		}
		if len(deleted) != 0 {
			t.Errorf("deleted %v in a dry run", deleted)
		}
	})

	t.Run("deletes only stale test objects", func(t *testing.T) {
		var deleted []string
		err := sweepCollection(context.Background(), newCollection(&deleted), false)
		if err == nil || !strings.Contains(err.Error(), "broken") {
			t.Errorf("sweepCollection error = %v, want the failed delete of broken", err)
		}
		slices.Sort(deleted)
		if want := []string{"broken", "gone", "stale"}; !slices.Equal(deleted, want) {
			t.Errorf("deleted %v, want %v", deleted, want)
		}
	})
}
