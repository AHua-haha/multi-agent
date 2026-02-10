package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"multi-agent/service"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type ViewFileArgs struct {
	File  string
	Lines [][]int
}
type EditFileArgs struct {
	File        string
	UnifiedDiff string
}
type CreateFileArgs struct {
	Path    string
	Content string
}
type BashArgs struct {
	Cmd string
	Dir string
}

func (s *Server) viewfileTool() (openai.FunctionDefinition, server.ToolHandlerFunc) {
	def := openai.FunctionDefinition{
		Name:        "view_file",
		Description: "Reads specific line ranges from a file. Use this to inspect code or text without loading the entire file into context. Multiple non-contiguous ranges can be requested at once.",
		Parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"File": {
					Type:        jsonschema.String,
					Description: "The full path to the file to be read.",
				},
				"Lines": {
					Type:        jsonschema.Array,
					Description: "A list of line ranges to retrieve. Each range is a pair of integers [start_line, end_line].",
					Items: &jsonschema.Definition{
						Type:        jsonschema.Array,
						Description: "A specific range defined as [start, end]. Line numbers are 1-indexed.",
						Items: &jsonschema.Definition{
							Type: jsonschema.Integer,
						},
					},
				},
			},
			Required: []string{"File", "Lines"},
		},
	}
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args ViewFileArgs
		err := request.BindArguments(&args)
		if err != nil {
			return nil, err
		}
		res, err := service.ViewFile(args.File, args.Lines)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(res), nil
	}
	return def, handler
}
func (s *Server) editfileTool() (openai.FunctionDefinition, server.ToolHandlerFunc) {
	def := openai.FunctionDefinition{
		Name:        "edit_file",
		Description: "Applies changes to a file by passing a unified diff to the 'patch' command-line utility. This is used for precise code modifications.",
		Parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"File": {
					Type:        jsonschema.String,
					Description: "The full path to the file to be patched.",
				},
				"UnifiedDiff": {
					Type:        jsonschema.String,
					Description: "The GNU unified diff string. It must include headers (---/+++) and hunk markers (@@). The diff will be piped to `patch -p1` or similar.",
				},
			},
			Required: []string{"File", "UnifiedDiff"},
		},
	}
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args EditFileArgs
		err := request.BindArguments(&args)
		if err != nil {
			return nil, err
		}
		err = service.EditFile(args.File, args.UnifiedDiff)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(fmt.Sprintf("Apply edit to file %s success", args.File)), nil
	}
	return def, handler
}
func (s *Server) createfileTool() (openai.FunctionDefinition, server.ToolHandlerFunc) {
	def := openai.FunctionDefinition{
		Name:        "create_file",
		Description: "Creates a new file at the specified path with the provided content. This function will fail and return an error if the file already exists to prevent overwriting.",
		Parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"Path": {
					Type:        jsonschema.String,
					Description: "The full path, including filename and extension, where the file should be created.",
				},
				"Content": {
					Type:        jsonschema.String,
					Description: "The full text content to be written to the new file.",
				},
			},
			Required: []string{"Path", "Content"},
		},
	}
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args CreateFileArgs
		err := request.BindArguments(&args)
		if err != nil {
			return nil, err
		}
		err = service.CreateFile(args.Path, args.Content)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(fmt.Sprintf("Create new file %s success", args.Path)), nil
	}
	return def, handler
}
func (s *Server) runbashTool() (openai.FunctionDefinition, server.ToolHandlerFunc) {
	def := openai.FunctionDefinition{
		Name:        "bash",
		Description: "Executes a bash command in a specified directory and returns the exit code and the output conbining stdout and stderr and a list of files/folders that were created, modified, or deleted during execution. Use this for running tests, build scripts, or system diagnostics.",
		Parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"Cmd": {
					Type:        jsonschema.String,
					Description: "The full bash command string to execute (e.g., 'go test ./...', 'ls -la', etc.).",
				},
				"Dir": {
					Type:        jsonschema.String,
					Description: "The working directory in which to execute the command. Defaults to the current project root if not specified.",
				},
			},
			Required: []string{"Cmd", "Dir"},
		},
	}
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args BashArgs
		err := request.BindArguments(&args)
		if err != nil {
			return nil, err
		}
		if args.Dir == "" {
			args.Dir = s.projectRoot
		}
		res, err := s.bashTool.Run(args.Cmd, args.Dir)
		if err != nil {
			return nil, err
		}
		resStr, _ := json.Marshal(res)
		return mcp.NewToolResultText(string(resStr)), nil
	}
	return def, handler
}
