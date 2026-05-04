package client

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// LoggingHTTPClient wraps an HTTP client and logs all requests and responses
// using terraform-plugin-log. It implements the HttpRequestDoer interface
// for the generated OpenAPI client.
type LoggingHTTPClient struct {
	client *http.Client
}

// NewLoggingHTTPClient creates a new LoggingHTTPClient that wraps the provided
// HTTP client.
func NewLoggingHTTPClient(client *http.Client) *LoggingHTTPClient {
	return &LoggingHTTPClient{
		client: client,
	}
}

// Do executes the HTTP request and logs the request path, response status code,
// and request duration. It extracts the context from the request to ensure logs
// are properly associated with the Terraform operation. A unique x-client-trace-id
// header is added to each request for traceability.
func (l *LoggingHTTPClient) Do(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	startTime := time.Now()

	// Generate and add x-client-trace-id header for request tracing
	requestID := uuid.New().String()
	req.Header.Set("x-client-trace-id", requestID)

	// Log the outgoing request
	tflog.Debug(ctx, "HTTP request", map[string]interface{}{
		"method":          req.Method,
		"path":            req.URL.Path,
		"url":             req.URL.String(),
		"client_trace_id": requestID,
	})

	// Execute the request
	resp, err := l.client.Do(req)
	duration := time.Since(startTime)

	// Log the response
	if err != nil {
		tflog.Error(ctx, "HTTP request failed", map[string]interface{}{
			"method":          req.Method,
			"path":            req.URL.Path,
			"error":           err.Error(),
			"duration_ms":     duration.Milliseconds(),
			"client_trace_id": requestID,
		})
		return resp, err
	}

	tflog.Debug(ctx, "HTTP response", map[string]interface{}{
		"method":          req.Method,
		"path":            req.URL.Path,
		"status_code":     resp.StatusCode,
		"duration_ms":     duration.Milliseconds(),
		"client_trace_id": requestID,
	})

	return resp, nil
}
