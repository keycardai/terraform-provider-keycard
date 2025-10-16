package client

import (
	"net/http"

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

// Do executes the HTTP request and logs the request path and response status code.
// It extracts the context from the request to ensure logs are properly associated
// with the Terraform operation.
func (l *LoggingHTTPClient) Do(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	// Log the outgoing request
	tflog.Debug(ctx, "HTTP request", map[string]interface{}{
		"method": req.Method,
		"path":   req.URL.Path,
		"url":    req.URL.String(),
	})

	// Execute the request
	resp, err := l.client.Do(req)

	// Log the response
	if err != nil {
		tflog.Error(ctx, "HTTP request failed", map[string]interface{}{
			"method": req.Method,
			"path":   req.URL.Path,
			"error":  err.Error(),
		})
		return resp, err
	}

	tflog.Debug(ctx, "HTTP response", map[string]interface{}{
		"method":      req.Method,
		"path":        req.URL.Path,
		"status_code": resp.StatusCode,
	})

	return resp, nil
}
