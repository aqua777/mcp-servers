module github.com/aqua777/mcp-servers/examples/time

go 1.25.8

replace github.com/aqua777/mcp-servers => ../../

replace github.com/aqua777/mcp-servers/examples/utils => ../utils

require (
	github.com/aqua777/mcp-servers/examples/utils v0.0.0-00010101000000-000000000000
	github.com/modelcontextprotocol/go-sdk v1.4.0
	github.com/sashabaranov/go-openai v1.41.2
)

require (
	github.com/google/jsonschema-go v0.4.2 // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.5.3 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	golang.org/x/oauth2 v0.34.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
)
