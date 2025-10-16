package client_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform-plugin-log/tflogtest"
	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

func TestLoggingHTTPClient(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/zones" {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"items":[],"page_info":{"has_next_page":false,"has_previous_page":false}}`))
			if err != nil {
				panic(err)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create a logging client
	loggingClient := client.NewLoggingHTTPClient(server.Client())

	// Create the API client with logging
	apiClient, err := client.NewClient(server.URL, client.WithHTTPClient(loggingClient))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	var output bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &output)

	// Make a request
	resp, err := apiClient.ListZones(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to list zones: %v", err)
	}
	defer resp.Body.Close()

	logEntries, err := tflogtest.MultilineJSONDecode(&output)
	if err != nil {
		t.Fatalf("Failed to decode log entries: %v", err)
	}

	if len(logEntries) != 2 {
		t.Fatalf("Expected 2 log entries got %d", len(logEntries))
	}

	// Define expected log entries
	wantRequestLog := map[string]interface{}{
		"@level":   "debug",
		"@message": "HTTP request",
		"method":   "GET",
		"path":     "/zones",
	}

	wantResponseLog := map[string]interface{}{
		"@level":      "debug",
		"@message":    "HTTP response",
		"method":      "GET",
		"path":        "/zones",
		"status_code": float64(200),
	}

	// Compare logs, ignoring dynamic fields like timestamps, url, and module
	ignoreFields := cmpopts.IgnoreMapEntries(func(k string, v interface{}) bool {
		return k == "@timestamp" || k == "url" || k == "@module"
	})

	if diff := cmp.Diff(wantRequestLog, logEntries[0], ignoreFields); diff != "" {
		t.Errorf("Request log mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(wantResponseLog, logEntries[1], ignoreFields); diff != "" {
		t.Errorf("Response log mismatch (-want +got):\n%s", diff)
	}

	// Verify url field exists in request log, the complete URL varies
	// per test as it depends on the httptest server address.
	if _, ok := logEntries[0]["url"]; !ok {
		t.Error("Expected request log to contain 'url' field")
	}
}

func TestLoggingHTTPClient_ErrorHandling(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte(`{"code":"internal_error","message":"Something went wrong","status":500}`))
		if err != nil {
			panic(err)
		}
	}))
	defer server.Close()

	// Create a logging client
	loggingClient := client.NewLoggingHTTPClient(server.Client())

	// Create the API client with logging
	apiClient, err := client.NewClient(server.URL, client.WithHTTPClient(loggingClient))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	var output bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &output)

	// Make a request that will return an error status
	resp, err := apiClient.ListZones(ctx, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	logEntries, err := tflogtest.MultilineJSONDecode(&output)
	if err != nil {
		t.Fatalf("Failed to decode log entries: %v", err)
	}

	if len(logEntries) != 2 {
		t.Fatalf("Expected 2 log entries got %d", len(logEntries))
	}

	// Define expected log entries
	wantRequestLog := map[string]interface{}{
		"@level":   "debug",
		"@message": "HTTP request",
		"method":   "GET",
		"path":     "/zones",
	}

	wantResponseLog := map[string]interface{}{
		"@level":      "debug",
		"@message":    "HTTP response",
		"method":      "GET",
		"path":        "/zones",
		"status_code": float64(500),
	}

	// Compare logs, ignoring dynamic fields like timestamps, url, and module
	ignoreFields := cmpopts.IgnoreMapEntries(func(k string, v interface{}) bool {
		return k == "@timestamp" || k == "url" || k == "@module"
	})

	if diff := cmp.Diff(wantRequestLog, logEntries[0], ignoreFields); diff != "" {
		t.Errorf("Request log mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(wantResponseLog, logEntries[1], ignoreFields); diff != "" {
		t.Errorf("Response log mismatch (-want +got):\n%s", diff)
	}
}

func TestLoggingHTTPClient_NetworkError(t *testing.T) {
	// Create a logging client with a client that will fail
	loggingClient := client.NewLoggingHTTPClient(http.DefaultClient)

	// Create the API client with logging
	apiClient, err := client.NewClient("http://localhost:1", client.WithHTTPClient(loggingClient))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	var output bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &output)

	// Make a request that will fail with a network error
	_, err = apiClient.ListZones(ctx, nil)
	if err == nil {
		t.Fatal("Expected network error, got nil")
	}

	logEntries, err := tflogtest.MultilineJSONDecode(&output)
	if err != nil {
		t.Fatalf("Failed to decode log entries: %v", err)
	}

	// Should have request log and error log
	if len(logEntries) != 2 {
		t.Fatalf("Expected 2 log entries got %d", len(logEntries))
	}

	// Define expected log entries
	wantRequestLog := map[string]interface{}{
		"@level":   "debug",
		"@message": "HTTP request",
		"method":   "GET",
		"path":     "/zones",
	}

	wantErrorLog := map[string]interface{}{
		"@level":   "error",
		"@message": "HTTP request failed",
		"method":   "GET",
		"path":     "/zones",
	}

	// Compare logs, ignoring dynamic fields like timestamps, url, module, and error message
	ignoreFields := cmpopts.IgnoreMapEntries(func(k string, v interface{}) bool {
		return k == "@timestamp" || k == "url" || k == "@module" || k == "error"
	})

	if diff := cmp.Diff(wantRequestLog, logEntries[0], ignoreFields); diff != "" {
		t.Errorf("Request log mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(wantErrorLog, logEntries[1], ignoreFields); diff != "" {
		t.Errorf("Error log mismatch (-want +got):\n%s", diff)
	}

	// Verify error field exists in error log, the actual error text varies
	// per test as it depends on the httptest server address.
	if _, ok := logEntries[1]["error"]; !ok {
		t.Error("Expected error log to contain 'error' field")
	}
}
