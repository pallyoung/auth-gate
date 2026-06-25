package routehost

import (
	"fmt"
	"net/netip"
	"strings"
)

const InvalidMessage = "host must be a hostname or IP address without scheme, port, or path"

func Normalize(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return ""
	}

	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		inner := host[1 : len(host)-1]
		if _, err := netip.ParseAddr(inner); err == nil {
			return inner
		}
	}

	return host
}

func IsValid(host string) bool {
	host = Normalize(host)
	if host == "" {
		return true
	}
	if strings.Contains(host, "://") || strings.ContainsAny(host, `/\?#@[]`) {
		return false
	}
	if _, err := netip.ParseAddr(host); err == nil {
		return true
	}
	if strings.Contains(host, ":") {
		return false
	}
	return isHostname(host)
}

func TLSListenHost(host string) string {
	return TLSListenHostPort(host, 443)
}

// TLSListenHostPort returns the listen address for a TLS server.
// Always binds to all interfaces (0.0.0.0) so the server is reachable
// from external clients. The host parameter is used for route matching,
// not for binding.
func TLSListenHostPort(host string, port int) string {
	portStr := fmt.Sprintf("%d", port)
	return ":" + portStr
}

func isHostname(host string) bool {
	if len(host) == 0 || len(host) > 253 {
		return false
	}

	labels := strings.Split(host, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, ch := range label {
			if (ch < 'a' || ch > 'z') && (ch < '0' || ch > '9') && ch != '-' {
				return false
			}
		}
	}

	return true
}
