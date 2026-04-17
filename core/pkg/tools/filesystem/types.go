package filesystem

const (
	FormatText = "text"
	FormatJSON = "json"

	ToolListDirectory          = "list_directory"
	ToolListDirectoryWithSizes = "list_directory_with_sizes"
	ToolDirectoryTree          = "directory_tree"
	ToolSearchFiles            = "search_files"
	ToolReadTextFile           = "read_text_file"
	ToolReadMediaFile          = "read_media_file"
	ToolReadMultipleFiles      = "read_multiple_files"
	ToolEditFile               = "edit_file"
	ToolGetFileInfo            = "get_file_info"
	ToolWriteFile              = "write_file"
	ToolCreateDirectory        = "create_directory"
	ToolMoveFile               = "move_file"
	ToolListAllowedDirectories = "list_allowed_directories"
	ToolGrep                   = "grep"
	ToolCopyFile               = "copy_file"
	ToolAppendFile             = "append_file"
	ToolCreateSymlink          = "create_symlink"
)

// Options holds server-level configuration.
type Options struct {
	// AllowedDirectories restricts all operations to paths within these directories.
	AllowedDirectories []string

	// OutputFormat sets the default output format for read operations.
	// Valid values: "text" (default), "json".
	// Can be overridden per-call via the "format" tool parameter.
	OutputFormat string

	// AIMode enables AI-first defaults: JSON output, structured errors.
	// When true, defaults to JSON format unless explicitly overridden.
	AIMode bool
}
