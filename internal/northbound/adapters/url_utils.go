package adapters

import (
	"net"
	"net/url"
	"strconv"
	"strings"
)

func normalizeServerURLWithPort(serverURL, protocol string, port int) string {
	serverURL = strings.TrimSpace(serverURL)
	if serverURL == "" {
		return ""
	}

	if !strings.Contains(serverURL, "://") {
		transport := strings.TrimSpace(protocol)
		if transport == "" {
			transport = "tcp"
		}
		serverURL = transport + "://" + serverURL
	}

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
