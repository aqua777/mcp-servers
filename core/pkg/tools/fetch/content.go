package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/go-shiori/go-readability"
)

func extractContentFromHTML(htmlContent string) (string, error) {
	article, err := readability.FromReader(strings.NewReader(htmlContent), nil)
	if err != nil || article.Content == "" {
		return "<error>Page failed to be simplified from HTML</error>", nil
	}

	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(article.Content)
	if err != nil {
		return "<error>Page failed to be simplified from HTML</error>", nil
	}

	return markdown, nil
}

func fetchURL(ctx context.Context, urlStr, userAgent string, forceRaw bool, proxyURL string) (content, prefix string, err error) {
	client, err := getHTTPClient(proxyURL)
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch %s: %w", urlStr, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", "", fmt.Errorf("failed to fetch %s - status code %d", urlStr, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response body: %w", err)
	}

	pageRaw := string(body)
	contentType := resp.Header.Get("Content-Type")

	isPageHTML := strings.Contains(pageRaw[:min(100, len(pageRaw))], "<html") ||
		strings.Contains(contentType, "text/html") ||
		contentType == ""

	if isPageHTML && !forceRaw {
		markdown, err := extractContentFromHTML(pageRaw)
		if err != nil {
			return pageRaw, fmt.Sprintf("Content type %s cannot be simplified to markdown, but here is the raw content:\n", contentType), nil
		}
		return markdown, "", nil
	}

	return pageRaw, fmt.Sprintf("Content type %s cannot be simplified to markdown, but here is the raw content:\n", contentType), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
