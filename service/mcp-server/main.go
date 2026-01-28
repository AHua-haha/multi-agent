package main

import (
	"context"
	"multi-agent/service"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	mgr := &service.TaskMgr{}
	s := server.NewMCPServer("Task Manager Mcp Server", "v1.0", server.WithToolCapabilities(true))
	s.AddTool(mcp.Tool{
		Name:        "ello",
		Description: "this is a tool",
	}, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return nil, nil
	})
	mgr.Config(s)
	err := server.ServeStdio(s)
	if err != nil {
		return
	}
}
