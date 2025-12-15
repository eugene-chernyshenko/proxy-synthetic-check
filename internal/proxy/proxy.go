package proxy

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/proxy"
)

// CreateTransport creates HTTP transport based on proxy protocol
func CreateTransport(protocol, proxyString string) (*http.Transport, error) {
	// Construct full URL from protocol + proxyString (proxyString contains username:password@host:port or host:port)
	proxyURL := protocol + "://" + proxyString
	proxyURI, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(protocol) {
	case "socks5":
		proxyAddr := proxyURI.Host
		if proxyAddr == "" {
			return nil, errors.New("proxy address (host:port) is not specified")
		}

		var auth *proxy.Auth
		if proxyURI.User != nil {
			password, _ := proxyURI.User.Password()
			auth = &proxy.Auth{
				User:     proxyURI.User.Username(),
				Password: password,
			}
		}

		dialer, err := proxy.SOCKS5("tcp", proxyAddr, auth, proxy.Direct)
		if err != nil {
			return nil, err
		}

		return &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			},
		}, nil

	case "http":
		// HTTP proxy using http.ProxyURL
		return &http.Transport{
			Proxy: http.ProxyURL(proxyURI),
		}, nil

	default:
		return nil, errors.New("unsupported proxy protocol: " + protocol)
	}
}

// MaskAuth hides password in URL for safe output
func MaskAuth(protocol, proxyString string) string {
	// Construct full URL for parsing
	fullURL := protocol + "://" + proxyString
	u, err := url.Parse(fullURL)
	if err != nil {
		return proxyString
	}
	if u.User != nil {
		u.User = url.User(u.User.Username())
	}
	return u.Host // Return just host:port without scheme for display
}

