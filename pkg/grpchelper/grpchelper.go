package grpchelper

import "strings"

func TransportSettings(apiEndpoint string) (string, bool) {
	// Insecure by default to preserve backwards compatibility
	insecure := true

	// Use TLS if explicitly asked or no schema is in the target
	if strings.Contains(apiEndpoint, "https://") || !strings.Contains(apiEndpoint, "://") {
		insecure = false
	}
	// sanitize but leave unix:// if presented
	target := strings.TrimPrefix(strings.TrimPrefix(apiEndpoint, "http://"), "https://")
	return target, insecure
}
