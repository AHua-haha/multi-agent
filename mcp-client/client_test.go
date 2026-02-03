package mcpclient

import (
	"context"
	"fmt"
	"log"
	mcpserver "multi-agent/mcp-server"
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
		args := mcpserver.BashArgs{
			Cmd: "pwd",
			// Dir: "/root/multi-agent/",
		}
		req.Params.Name = "bash"
		req.Params.Arguments = args
		res, err := mcpClient.CallTool(ctx, req)
		fmt.Printf("%v\n", res.Content)
	})
}
