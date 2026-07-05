package domain

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ParseEndpoint accepts local paths, scp-style SSH endpoints and ssh:// URLs.
func ParseEndpoint(value string) (Endpoint, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return Endpoint{}, fmt.Errorf("endpoint is required")
	}
	if strings.HasPrefix(value, "ssh://") {
		parsed, err := url.Parse(value)
		if err != nil {
			return Endpoint{}, err
		}
		port := 0
		if parsed.Port() != "" {
			port, err = strconv.Atoi(parsed.Port())
			if err != nil {
				return Endpoint{}, err
			}
		}
		user := ""
		if parsed.User != nil {
			user = parsed.User.Username()
		}
		endpoint := Endpoint{
			Kind: EndpointSSH,
			Host: parsed.Hostname(),
			User: user,
			Port: port,
			Path: parsed.Path,
		}
		return endpoint, endpoint.Validate()
	}
	if colon := strings.Index(value, ":"); colon > 0 && !strings.HasPrefix(value, "/") {
		remote := value[:colon]
		path := value[colon+1:]
		user, host, _ := strings.Cut(remote, "@")
		if host == "" {
			host = user
			user = ""
		}
		endpoint := Endpoint{Kind: EndpointSSH, Host: host, User: user, Path: path}
		return endpoint, endpoint.Validate()
	}
	endpoint := Endpoint{Kind: EndpointLocal, Path: value}
	return endpoint, endpoint.Validate()
}
