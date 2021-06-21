package grpchelper

import "strings"

func TransportSettings(apiEndpoint string) (string, bool) {
	// HTTP is always insecure
	if strings.HasPrefix(apiEndpoint, "http://") {
		return strings.TrimPrefix(apiEndpoint, "http://"), true
	}

	// Unix domain sockets are always insecure
	if strings.HasPrefix(apiEndpoint, "unix:") {
		return apiEndpoint, true
	}

	// HTTPS and other cases are always secure
	return strings.TrimPrefix(apiEndpoint, "https://"), false
}
