package mcpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestMcpClient(t *testing.T) {
	t.Run("test client", func(t *testing.T) {
		ctx := context.Background()

		mcpClient, err := client.NewStdioMCPClient("/root/multi-agent/bin/mcpserver", nil)
		if err != nil {
			log.Fatalf("Failed to create client: %v", err)
		}
		defer mcpClient.Close()
		err = mcpClient.Start(ctx)
		_, err = mcpClient.Initialize(ctx, mcp.InitializeRequest{
			Params: mcp.InitializeParams{
				ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
				ClientInfo: mcp.Implementation{
					Name:    "sampling-example-client",
					Version: "1.0.0",
				},
				Capabilities: mcp.ClientCapabilities{},
			},
		})
		req := mcp.CallToolRequest{}
		// args := mcpserver.BashArgs{
		// 	Cmd: "pwd",
		// 	// Dir: "/root/multi-agent/",
		// }
		args := `{"Cmd": "ls -la /root/multi-agent"}`
		req.Params.Name = "bash"
		req.Params.Arguments = json.RawMessage(args)
		res, err := mcpClient.CallTool(ctx, req)
		if err != nil {
			fmt.Printf("err: %v\n", err)
		}
		fmt.Printf("%v\n", res.Content)
	})
}
func TestLspMcp(t *testing.T) {
	t.Run("test client", func(t *testing.T) {
		ctx := context.Background()

		mcpClient, err := client.NewStdioMCPClient("uv", nil, "run", "/root/multi-agent/cmds/fileMCP/lsp-mcp.py")
		if err != nil {
			log.Fatalf("Failed to create client: %v", err)
		}
		defer mcpClient.Close()
		err = mcpClient.Start(ctx)
		_, err = mcpClient.Initialize(ctx, mcp.InitializeRequest{
			Params: mcp.InitializeParams{
				ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
				ClientInfo: mcp.Implementation{
					Name:    "sampling-example-client",
					Version: "1.0.0",
				},
				Capabilities: mcp.ClientCapabilities{},
			},
		})
		res, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
		for _, t := range res.Tools {
			byte, _ := json.MarshalIndent(t, "", "  ")
			fmt.Printf("%s\n", byte)
		}
		para := mcp.CallToolParams{
			Name: "request_document_symbols",
			Arguments: map[string]any{
				"relative_path": "/root/multi-agent/mcp-server/tool_def.go",
			},
		}
		toolres, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: para,
		})
		fmt.Printf("toolres.Content: %v\n", toolres.Content)
	})
}
