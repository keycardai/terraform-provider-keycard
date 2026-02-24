//go:generate go tool oapi-codegen -config ../../configs/oapi/client.yaml ../../api/openapi.yaml
package client

// Endpoint returns the server base URL that this client was configured with.
// It extracts the Server field from the underlying Client implementation.
func (c *ClientWithResponses) Endpoint() string {
	if inner, ok := c.ClientInterface.(*Client); ok {
		return inner.Server
	}
	return ""
}
