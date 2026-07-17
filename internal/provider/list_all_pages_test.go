package provider

import (
	"fmt"
	"testing"

	"github.com/oapi-codegen/nullable"
)

func TestListAllPages_walksAllPages(t *testing.T) {
	pages := [][]string{{"a", "b"}, {"c"}, {"d", "e"}}
	var requests []string

	items, err := listAllPages(func(after *string) ([]string, nullable.Nullable[string], error) {
		page := 0
		if after != nil {
			if _, err := fmt.Sscanf(*after, "cursor-%d", &page); err != nil {
				t.Fatalf("unexpected after cursor %q", *after)
			}
			requests = append(requests, *after)
		} else {
			requests = append(requests, "")
		}

		cursor := nullable.NewNullNullable[string]()
		if page+1 < len(pages) {
			cursor = nullable.NewNullableWithValue(fmt.Sprintf("cursor-%d", page+1))
		}
		return pages[page], cursor, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	want := []string{"a", "b", "c", "d", "e"}
	if len(items) != len(want) {
		t.Fatalf("expected %d items, got %d: %v", len(want), len(items), items)
	}
	for i, w := range want {
		if items[i] != w {
			t.Fatalf("expected items %v, got %v", want, items)
		}
	}
	if len(requests) != 3 {
		t.Fatalf("expected 3 page requests, got %d: %v", len(requests), requests)
	}
}

func TestListAllPages_absentCursorStops(t *testing.T) {
	calls := 0
	items, err := listAllPages(func(after *string) ([]string, nullable.Nullable[string], error) {
		calls++
		return nil, nullable.Nullable[string]{}, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected no items, got %v", items)
	}
	if calls != 1 {
		t.Fatalf("expected a single page request, got %d", calls)
	}
}

func TestListAllPages_propagatesError(t *testing.T) {
	_, err := listAllPages(func(after *string) ([]string, nullable.Nullable[string], error) {
		return nil, nullable.Nullable[string]{}, fmt.Errorf("boom")
	})
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected fetch error to propagate, got %v", err)
	}
}
