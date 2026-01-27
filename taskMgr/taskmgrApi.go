package taskMgr

import (
	"context"
	"multi-agent/shared"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func (mgr *TaskMgr) Config(s *server.MCPServer) {
	mgr.registerCreate(s)
	mgr.registerFinish(s)
}

func (mgr *TaskMgr) registerCreate(s *server.MCPServer) {
	def := openai.FunctionDefinition{
		Name:        "create_task",
		Description: "",
		Parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"answer": {
					Type:        jsonschema.String,
					Description: "",
				},
			},
		},
	}
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args shared.TaskCreateArgs
		err := request.BindArguments(&args)
		if err != nil {
			return nil, err
		}
		var res string
		return mcp.NewToolResultText(res), nil
	}
	tool, err := shared.ConvertToMcpTool(def)
	if err != nil {
		return
	}
	s.AddTool(tool, handler)
}
func (mgr *TaskMgr) registerFinish(s *server.MCPServer) {
	def := openai.FunctionDefinition{
		Name:        "finish_task",
		Description: "",
		Parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"answer": {
					Type:        jsonschema.String,
					Description: "",
				},
			},
		},
	}
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args shared.TaskFinishedArgs
		err := request.BindArguments(&args)
		if err != nil {
			return nil, err
		}
		var res string
		return mcp.NewToolResultText(res), nil
	}
	tool, err := shared.ConvertToMcpTool(def)
	if err != nil {
		return
	}
	s.AddTool(tool, handler)
}
