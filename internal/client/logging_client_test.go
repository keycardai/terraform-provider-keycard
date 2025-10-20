package client_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
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

	// Compare logs, ignoring dynamic fields like timestamps, url, module, duration, and client_trace_id
	ignoreFields := cmpopts.IgnoreMapEntries(func(k string, v interface{}) bool {
		return k == "@timestamp" || k == "url" || k == "@module" || k == "duration_ms" || k == "client_trace_id"
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

	// Verify client_trace_id field exists in both request and response logs
	if _, ok := logEntries[0]["client_trace_id"]; !ok {
		t.Error("Expected request log to contain 'client_trace_id' field")
	}
	if _, ok := logEntries[1]["client_trace_id"]; !ok {
		t.Error("Expected response log to contain 'client_trace_id' field")
	}

	// Verify client_trace_id is consistent between request and response logs
	if logEntries[0]["client_trace_id"] != logEntries[1]["client_trace_id"] {
		t.Errorf("Request ID mismatch: request=%v, response=%v", logEntries[0]["client_trace_id"], logEntries[1]["client_trace_id"])
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

	// Compare logs, ignoring dynamic fields like timestamps, url, module, duration, and client_trace_id
	ignoreFields := cmpopts.IgnoreMapEntries(func(k string, v interface{}) bool {
		return k == "@timestamp" || k == "url" || k == "@module" || k == "duration_ms" || k == "client_trace_id"
	})

	if diff := cmp.Diff(wantRequestLog, logEntries[0], ignoreFields); diff != "" {
		t.Errorf("Request log mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(wantResponseLog, logEntries[1], ignoreFields); diff != "" {
		t.Errorf("Response log mismatch (-want +got):\n%s", diff)
	}

	// Verify client_trace_id field exists in both request and response logs
	if _, ok := logEntries[0]["client_trace_id"]; !ok {
		t.Error("Expected request log to contain 'client_trace_id' field")
	}
	if _, ok := logEntries[1]["client_trace_id"]; !ok {
		t.Error("Expected response log to contain 'client_trace_id' field")
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

	// Compare logs, ignoring dynamic fields like timestamps, url, module, error message, duration, and client_trace_id
	ignoreFields := cmpopts.IgnoreMapEntries(func(k string, v interface{}) bool {
		return k == "@timestamp" || k == "url" || k == "@module" || k == "error" || k == "duration_ms" || k == "client_trace_id"
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

	// Verify client_trace_id field exists in both request and error logs
	if _, ok := logEntries[0]["client_trace_id"]; !ok {
		t.Error("Expected request log to contain 'client_trace_id' field")
	}
	if _, ok := logEntries[1]["client_trace_id"]; !ok {
		t.Error("Expected error log to contain 'client_trace_id' field")
	}
}

func TestLoggingHTTPClient_RequestIDHeader(t *testing.T) {
	var capturedHeaders []http.Header

	// Create a test server that captures request headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = append(capturedHeaders, r.Header.Clone())
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"items":[],"page_info":{"has_next_page":false,"has_previous_page":false}}`))
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

	ctx := context.Background()

	// Make first request
	resp1, err := apiClient.ListZones(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to list zones (request 1): %v", err)
	}
	defer resp1.Body.Close()

	// Make second request
	resp2, err := apiClient.ListZones(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to list zones (request 2): %v", err)
	}
	defer resp2.Body.Close()

	// Verify we captured headers from both requests
	if len(capturedHeaders) != 2 {
		t.Fatalf("Expected 2 sets of captured headers, got %d", len(capturedHeaders))
	}

	// Verify x-client-trace-id header exists in first request
	requestID1 := capturedHeaders[0].Get("x-client-trace-id")
	if requestID1 == "" {
		t.Error("Expected x-client-trace-id header in first request, got empty string")
	}

	// Verify x-client-trace-id header exists in second request
	requestID2 := capturedHeaders[1].Get("x-client-trace-id")
	if requestID2 == "" {
		t.Error("Expected x-client-trace-id header in second request, got empty string")
	}

	// Verify the request IDs are different (each request should get a unique ID)
	if requestID1 == requestID2 {
		t.Errorf("Expected different request IDs for different requests, both were: %s", requestID1)
	}

	// Verify the request IDs are valid UUIDs
	if _, err := uuid.Parse(requestID1); err != nil {
		t.Errorf("Expected request ID 1 to be UUID: %s", err)
	}
	if _, err := uuid.Parse(requestID2); err != nil {
		t.Errorf("Expected request ID 2 to be UUID: %s", err)
	}
}
