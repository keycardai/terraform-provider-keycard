package provider

import (
	"fmt"
	"net/url"
	"strings"
)

// deriveServiceURL replaces the "api." subdomain prefix of an API endpoint
// with the given subdomain. For example:
//
//	deriveServiceURL("https://api.keycard.ai", "id") => "https://id.keycard.ai"
//	deriveServiceURL("https://api.staging.keycard.ai", "console") => "https://console.staging.keycard.ai"
func deriveServiceURL(apiEndpoint, subdomain string) (string, error) {
	u, err := url.Parse(apiEndpoint)
	if err != nil {
		return "", fmt.Errorf("invalid API endpoint %q: %w", apiEndpoint, err)
	}

	host := u.Hostname()
	port := u.Port()

	if !strings.HasPrefix(host, "api.") {
		return "", fmt.Errorf("cannot derive %s URL from endpoint %q: host does not start with \"api.\"", subdomain, apiEndpoint)
	}

	host = subdomain + "." + strings.TrimPrefix(host, "api.")

	if port != "" {
		u.Host = host + ":" + port
	} else {
		u.Host = host
	}

	u.Path = ""
	u.RawQuery = ""
	u.Fragment = ""

	return u.String(), nil
}

// identityURL derives the identity service URL from the API endpoint.
// e.g., "https://api.keycard.ai" -> "https://id.keycard.ai"
func identityURL(apiEndpoint string) (string, error) {
	return deriveServiceURL(apiEndpoint, "id")
}

// consoleURL derives the console URL from the API endpoint.
// e.g., "https://api.keycard.ai" -> "https://console.keycard.ai"
func consoleURL(apiEndpoint string) (string, error) {
	return deriveServiceURL(apiEndpoint, "console")
}
