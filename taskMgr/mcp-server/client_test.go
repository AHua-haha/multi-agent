package main

import (
	"context"
	"log"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestMcpClient(t *testing.T) {
	t.Run("test client", func(t *testing.T) {
		ctx := context.Background()

		mcpClient, err := client.NewStdioMCPClient("/root/workspace/multi-agent/bin/mcp-server", nil)
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
		toolsResult, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
		if err != nil {
			log.Fatalf("Failed to list tools: %v", err)
		}

		log.Printf("Available tools:")
		for _, tool := range toolsResult.Tools {
			log.Printf("  - %s: %s", tool.Name, tool.Description)
		}
	})
}
