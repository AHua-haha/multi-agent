package main

import (
	"log"
	"net/http"

	"github.com/mark3labs/mcp-go/server"
	mcpserver "multi-agent/mcp-server"
)

func main() {
	s := server.NewMCPServer("calculator-server", "1.0.0")

	s.AddTool(mcpserver.CalculatorTool, mcpserver.HandleCalculator)

	sseServer := server.NewSSEServer(s)
	http.Handle("/mcp", sseServer)

	log.Println("MCP Calculator Server running on http://localhost:8080/mcp")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
