package fetch

const (
	DefaultUserAgentAutonomous = "ModelContextProtocol/1.0 (Autonomous; +https://github.com/modelcontextprotocol/servers)"
	DefaultUserAgentManual     = "ModelContextProtocol/1.0 (User-Specified; +https://github.com/modelcontextprotocol/servers)"
)

type Options struct {
	CustomUserAgent string
	IgnoreRobotsTxt bool
	ProxyURL        string
}

type FetchArgs struct {
	URL        string `json:"url" jsonschema:"required,description=URL to fetch"`
	MaxLength  int    `json:"max_length,omitempty" jsonschema:"description=Maximum number of characters to return,default=5000,minimum=1,maximum=1000000"`
	StartIndex int    `json:"start_index,omitempty" jsonschema:"description=On return output starting at this character index,default=0,minimum=0"`
	Raw        bool   `json:"raw,omitempty" jsonschema:"description=Get the actual HTML content without simplification,default=false"`
}
