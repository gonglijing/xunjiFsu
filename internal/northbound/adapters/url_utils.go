package adapters

import (
	"net"
	"net/url"
	"strconv"
	"strings"
)

func normalizeServerURLWithPort(serverURL, protocol string, port int) string {
	serverURL = ensureServerURLProtocol(serverURL, protocol)
	if serverURL == "" {
		return ""
	}
	return appendServerURLPort(serverURL, port)
}

func ensureServerURLProtocol(serverURL, protocol string) string {
	trimmedServerURL := strings.TrimSpace(serverURL)
	if trimmedServerURL == "" {
		return ""
	}
	if strings.Contains(trimmedServerURL, "://") {
		return trimmedServerURL
	}

	transport := strings.TrimSpace(protocol)
	if transport == "" {
		transport = "tcp"
	}
	return transport + "://" + trimmedServerURL
}

func appendServerURLPort(serverURL string, port int) string {
	if port <= 0 {
		return serverURL
	}

	parsed, err := url.Parse(serverURL)
	if err != nil {
		return serverURL
	}
	if parsed.Port() != "" {
		return serverURL
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		return serverURL
	}

	parsed.Host = net.JoinHostPort(hostname, strconv.Itoa(port))
	return parsed.String()
}
