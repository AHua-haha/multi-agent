package taskMgr

import (
	"context"
	"multi-agent/shared"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sashabaranov/go-openai"
)

func (mgr *TaskMgr) Config(s *server.MCPServer) {

}

func (mgr *TaskMgr) registerCreate(s *server.MCPServer) {
	def := openai.FunctionDefinition{}
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
