package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

// pagedNames serves GET list endpoints returning names one per page, chained
// by after_cursor, mimicking svc-pdp's substring query[name] semantics: only
// names containing the query substring are included in the walk.
type pagedNames struct {
	names    []string
	requests []string
}

func (p *pagedNames) handler(t *testing.T, itemsKey func(name string) map[string]any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		p.requests = append(p.requests, r.URL.RawQuery)

		var filtered []string
		for _, n := range p.names {
			if containsFold(n, q.Get("query[name]")) {
				filtered = append(filtered, n)
			}
		}

		page := 0
		if after := q.Get("after"); after != "" {
			if _, err := fmt.Sscanf(after, "cursor-%d", &page); err != nil {
				t.Errorf("unexpected after cursor %q", after)
			}
		}

		var items []map[string]any
		var cursor any
		if page < len(filtered) {
			items = []map[string]any{itemsKey(filtered[page])}
			if page+1 < len(filtered) {
				cursor = fmt.Sprintf("cursor-%d", page+1)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"items":      items,
			"pagination": map[string]any{"after_cursor": cursor, "before_cursor": nil},
		}); err != nil {
			t.Errorf("encoding response: %s", err)
		}
	}
}

func containsFold(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func testClientFor(t *testing.T, srv *httptest.Server) *client.ClientWithResponses {
	t.Helper()
	c, err := client.NewClientWithResponses(srv.URL)
	if err != nil {
		t.Fatalf("building client: %s", err)
	}
	return c
}

func policySetItem(name string) map[string]any {
	return map[string]any{
		"id":          "psid-" + name,
		"zone_id":     "z1",
		"name":        name,
		"target_type": "zone",
		"owner_type":  "customer",
		"created_by":  "test",
		"created_at":  time.Time{}.Format(time.RFC3339),
		"updated_at":  time.Time{}.Format(time.RFC3339),
		"active":      false,
	}
}

func policyItem(name string) map[string]any {
	return map[string]any{
		"id":         "pid-" + name,
		"zone_id":    "z1",
		"name":       name,
		"owner_type": "customer",
		"created_by": "test",
	}
}

func TestFindPolicySetsByName_exactMatchBeyondFirstPage(t *testing.T) {
	// Substring cousins sort ahead of the exact match, so "prod" only appears
	// on the final page of the query[name]=prod walk.
	backend := &pagedNames{names: []string{"production-a", "PRODUCT-b", "prod"}}
	srv := httptest.NewServer(backend.handler(t, policySetItem))
	defer srv.Close()

	matches, err := findPolicySetsByName(context.Background(), testClientFor(t, srv), "z1", "prod")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(matches) != 1 || matches[0].Name != "prod" {
		t.Fatalf("expected exactly the exact-name match, got %+v", matches)
	}
	if len(backend.requests) != 3 {
		t.Fatalf("expected 3 paginated requests, got %d: %v", len(backend.requests), backend.requests)
	}
}

func TestFindPolicySetsByName_noMatchExhaustsCursor(t *testing.T) {
	backend := &pagedNames{names: []string{"production-a", "production-b"}}
	srv := httptest.NewServer(backend.handler(t, policySetItem))
	defer srv.Close()

	matches, err := findPolicySetsByName(context.Background(), testClientFor(t, srv), "z1", "prod")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(matches) != 0 {
		t.Fatalf("expected no matches, got %+v", matches)
	}
	if len(backend.requests) != 2 {
		t.Fatalf("expected full cursor walk (2 requests), got %d", len(backend.requests))
	}
}

func TestFindPolicySetsByName_duplicateStopsEarly(t *testing.T) {
	backend := &pagedNames{names: []string{"prod", "prod", "prod"}}
	srv := httptest.NewServer(backend.handler(t, policySetItem))
	defer srv.Close()

	matches, err := findPolicySetsByName(context.Background(), testClientFor(t, srv), "z1", "prod")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(matches) != 2 {
		t.Fatalf("expected early stop at 2 matches, got %d", len(matches))
	}
	if len(backend.requests) != 2 {
		t.Fatalf("expected 2 requests before early stop, got %d", len(backend.requests))
	}
}

func TestFindPoliciesByName_exactMatchBeyondFirstPage(t *testing.T) {
	backend := &pagedNames{names: []string{"production-a", "PRODUCT-b", "prod"}}
	srv := httptest.NewServer(backend.handler(t, policyItem))
	defer srv.Close()

	matches, err := findPoliciesByName(context.Background(), testClientFor(t, srv), "z1", "prod")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(matches) != 1 || matches[0].Name != "prod" {
		t.Fatalf("expected exactly the exact-name match, got %+v", matches)
	}
	if len(backend.requests) != 3 {
		t.Fatalf("expected 3 paginated requests, got %d: %v", len(backend.requests), backend.requests)
	}
}
