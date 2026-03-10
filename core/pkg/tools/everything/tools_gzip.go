package everything

import (
"bytes"
"compress/gzip"
"context"
"encoding/base64"
"encoding/json"
"fmt"
"io"
"net/http"
"net/url"
"os"
"strconv"
"strings"
"time"

"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
gzipMaxFetchSize       = 10 * 1024 * 1024
gzipMaxFetchTimeMillis = 30000
gzipAllowedDomains     = []string{}
)

func initGzipConfig() {
	if s := os.Getenv("GZIP_MAX_FETCH_SIZE"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			gzipMaxFetchSize = v
		}
	}
	if s := os.Getenv("GZIP_MAX_FETCH_TIME_MILLIS"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			gzipMaxFetchTimeMillis = v
		}
	}
	if s := os.Getenv("GZIP_ALLOWED_DOMAINS"); s != "" {
		parts := strings.Split(s, ",")
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				gzipAllowedDomains = append(gzipAllowedDomains, strings.ToLower(trimmed))
			}
		}
	}
}

func registerGZipFileAsResourceTool(server *mcp.Server) {
	initGzipConfig()

	server.AddTool(&mcp.Tool{
		Name:        "gzip-file-as-resource",
		Description: "Compresses a single file using gzip compression. Depending upon the selected output type, returns either the compressed data as a gzipped resource or a resource link, allowing it to be downloaded in a subsequent request during the current session.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "Name of the output file",
					"default":     "README.md.gz",
				},
				"data": map[string]any{
					"type":        "string",
					"format":      "uri",
					"description": "URL or data URI of the file content to compress",
					"default":     "https://raw.githubusercontent.com/modelcontextprotocol/servers/refs/heads/main/README.md",
				},
				"outputType": map[string]any{
					"type":        "string",
					"enum":        []string{"resourceLink", "resource"},
					"description": "How the resulting gzipped file should be returned. 'resourceLink' returns a link to a resource that can be read later, 'resource' returns a full resource object.",
					"default":     "resourceLink",
				},
			},
			"required": []string{},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Name       string `json:"name"`
			Data       string `json:"data"`
			OutputType string `json:"outputType"`
		}
		
		// Set defaults
		args.Name = "README.md.gz"
		args.Data = "https://raw.githubusercontent.com/modelcontextprotocol/servers/refs/heads/main/README.md"
		args.OutputType = "resourceLink"

		if len(request.Params.Arguments) > 0 {
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				return handleError(err)
			}
		}

		u, err := url.Parse(args.Data)
		if err != nil {
			return handleError(fmt.Errorf("invalid URL: %w", err))
		}

		if u.Scheme != "http" && u.Scheme != "https" {
			return handleError(fmt.Errorf("unsupported URL protocol for %s. Only http and https URLs are supported", args.Data))
		}

		if len(gzipAllowedDomains) > 0 {
			domain := u.Hostname()
			allowed := false
			for _, d := range gzipAllowedDomains {
				if domain == d || strings.HasSuffix(domain, "."+d) {
					allowed = true
					break
				}
			}
			if !allowed {
				return handleError(fmt.Errorf("domain %s is not in the allowed domains list", domain))
			}
		}

		ctx, cancel := context.WithTimeout(ctx, time.Duration(gzipMaxFetchTimeMillis)*time.Millisecond)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, args.Data, nil)
		if err != nil {
			return handleError(err)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return handleError(fmt.Errorf("failed to fetch data: %w", err))
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return handleError(fmt.Errorf("failed to fetch data: HTTP %d", resp.StatusCode))
		}

		// Read data with limit
		bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, int64(gzipMaxFetchSize+1)))
		if err != nil {
			return handleError(err)
		}
		if len(bodyBytes) > gzipMaxFetchSize {
			return handleError(fmt.Errorf("response from %s exceeds %d bytes", args.Data, gzipMaxFetchSize))
		}

		var compressedBuffer bytes.Buffer
		gz := gzip.NewWriter(&compressedBuffer)
		if _, err := gz.Write(bodyBytes); err != nil {
			return handleError(fmt.Errorf("failed to compress data: %w", err))
		}
		if err := gz.Close(); err != nil {
			return handleError(fmt.Errorf("failed to compress data: %w", err))
		}

		uri := fmt.Sprintf("demo://resource/session/%s", args.Name)
		blob := compressedBuffer.Bytes()
		blobBase64 := base64.StdEncoding.EncodeToString(blob)
		mimeType := "application/gzip"

		// Register the resource if not already registered (we use a simple dynamic registration here)
		// NOTE: In the TS implementation, this uses a "session" scope which is only valid for that session.
		// Since the Go SDK doesn't have a direct equivalent to session-scoped resources yet, we register it
// globally for the server. In a production app, we would manage this per-session.
server.AddResource(&mcp.Resource{
URI:         uri,
Name:        args.Name,
MIMEType:    mimeType,
Description: fmt.Sprintf("Gzipped content of %s", args.Data),
}, func(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
return &mcp.ReadResourceResult{
Contents: []*mcp.ResourceContents{
{
URI:      uri,
MIMEType: mimeType,
Blob:     blob,
},
},
}, nil
})

var content mcp.Content
if args.OutputType == "resource" {
// To return a full resource inline
resContent := map[string]interface{}{
"type": "resource",
"resource": map[string]interface{}{
"uri":      uri,
"mimeType": mimeType,
"blob":     blobBase64,
},
}
b, _ := json.Marshal(resContent)
content = &mcp.TextContent{Text: string(b)}
} else {
// Return a resourceLink
resLink := map[string]interface{}{
"type":     "resource_link",
"uri":      uri,
"name":     args.Name,
"mimeType": mimeType,
}
b, _ := json.Marshal(resLink)
content = &mcp.TextContent{Text: string(b)}
}

return &mcp.CallToolResult{
Content: []mcp.Content{content},
}, nil
})
}
