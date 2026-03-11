package fetch

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// HTTPClientFactory allows overriding the default HTTP client for testing.
var HTTPClientFactory = func(proxyURL string) (*http.Client, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	if proxyURL != "" {
		proxyURLParsed, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}
		client.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURLParsed),
		}
	}

	return client, nil
}

func getHTTPClient(proxyURL string) (*http.Client, error) {
	return HTTPClientFactory(proxyURL)
}
