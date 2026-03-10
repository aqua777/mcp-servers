package everything

import (
	"context"
	_ "embed"

	"github.com/aqua777/mcp-servers/common"
	"github.com/aqua777/mcp-servers/core/pkg/runtime"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed instructions.md
var serverInstructions string

func init() {
	runtime.Register(common.MCP_Everything, NewServer)
}

// Options captures future configuration for the Everything server. Placeholder until
// feature parity work lands.
type Options struct {
	GzipMaxFetchSize       int
	GzipMaxFetchTimeMillis int
	GzipAllowedDomains     string
}

// NewServer builds a minimal MCP server placeholder so the Everything Go binary can
// be scaffolded before full feature parity is implemented.
func NewServer(ctx context.Context, opts any) (*mcp.Server, error) {
	_ = ctx
	_ = opts

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-everything",
		Version: "2.0.0",
	}, &mcp.ServerOptions{
		Instructions: serverInstructions,
	})

	registerTools(server)
	registerGetRootsListTool(server)
	registerTriggerElicitationRequestTool(server)
	registerTriggerSamplingRequestTool(server)
	registerGZipFileAsResourceTool(server)

	return server, nil
}
