package resources

import (
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	ResourceTypeBlob = "blob"
	ResourceTypeText = "text"
)

func TextResourceUri(id int) string {
	return fmt.Sprintf("demo://resource/dynamic/text/%d", id)
}

func BlobResourceUri(id int) string {
	return fmt.Sprintf("demo://resource/dynamic/blob/%d", id)
}

func TextResource(uri string, id int) mcp.ResourceContents {
	return mcp.ResourceContents{
		URI:      uri,
		MIMEType: "text/plain",
		Text:     fmt.Sprintf("This is dynamically generated text resource %d.\nGenerated at: %s", id, time.Now().Format(time.RFC3339)),
	}
}

func BlobResource(uri string, id int) mcp.ResourceContents {
	data := fmt.Sprintf("This is dynamically generated blob resource %d.\nGenerated at: %s", id, time.Now().Format(time.RFC3339))
	return mcp.ResourceContents{
		URI:      uri,
		MIMEType: "application/octet-stream",
		Blob:     []byte(data),
	}
}
