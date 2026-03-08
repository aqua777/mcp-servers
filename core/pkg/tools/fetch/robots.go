package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/temoto/robotstxt"
)

func getRobotsTxtURL(urlStr string) (string, error) {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	robotsURL := &url.URL{
		Scheme: parsed.Scheme,
		Host:   parsed.Host,
		Path:   "/robots.txt",
	}

	return robotsURL.String(), nil
}

func checkMayAutonomouslyFetchURL(ctx context.Context, urlStr, userAgent, proxyURL string) error {
	robotsURL, err := getRobotsTxtURL(urlStr)
	if err != nil {
		return fmt.Errorf("failed to construct robots.txt URL: %w", err)
	}

	client := &http.Client{}
	if proxyURL != "" {
		proxyURLParsed, err := url.Parse(proxyURL)
		if err != nil {
			return fmt.Errorf("invalid proxy URL: %w", err)
		}
		client.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURLParsed),
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", robotsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request for robots.txt: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch robots.txt %s due to a connection issue: %w", robotsURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("when fetching robots.txt (%s), received status %d so assuming that autonomous fetching is not allowed, the user can try manually fetching by using the fetch prompt",
			robotsURL, resp.StatusCode)
	}

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return nil
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read robots.txt: %w", err)
		}

		robotsTxt := string(body)
		processedRobotsTxt := processRobotsTxt(robotsTxt)

		robots, err := robotstxt.FromBytes([]byte(processedRobotsTxt))
		if err != nil {
			return nil
		}

		if !robots.TestAgent(urlStr, userAgent) {
			return fmt.Errorf("the sites robots.txt (%s), specifies that autonomous fetching of this page is not allowed, "+
				"<useragent>%s</useragent>\n"+
				"<url>%s</url>"+
				"<robots>\n%s\n</robots>\n"+
				"The assistant must let the user know that it failed to view the page. The assistant may provide further guidance based on the above information.\n"+
				"The assistant can tell the user that they can try manually fetching the page by using the fetch prompt within their UI.",
				robotsURL, userAgent, urlStr, robotsTxt)
		}
	}

	return nil
}

func processRobotsTxt(robotsTxt string) string {
	var lines []string
	for _, line := range strings.Split(robotsTxt, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}
