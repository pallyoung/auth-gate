package routehost

import (
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
	host = Normalize(host)
	if host == "" {
		return ":443"
	}
	if addr, err := netip.ParseAddr(host); err == nil && addr.Is6() {
		return "[" + host + "]:443"
	}
	if strings.Contains(host, ":") {
		return host
	}
	return host + ":443"
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
