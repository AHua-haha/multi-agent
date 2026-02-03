package mcpserver

import (
	"multi-agent/service"
	"multi-agent/shared"

	"github.com/mark3labs/mcp-go/server"
	"github.com/sashabaranov/go-openai"
)

type Server struct {
	projectRoot string
	bashTool    service.BashTool
	mcpServer   *server.MCPServer
}

func NewServer(projectRoot string) (*Server, error) {
	s := &Server{
		projectRoot: projectRoot,
		mcpServer:   server.NewMCPServer("file editing and bash", "v1.0", server.WithToolCapabilities(true)),
	}
	err := s.bashTool.AddRepo(projectRoot)
	if err != nil {
		return nil, err
	}
	tools := []func() (openai.FunctionDefinition, server.ToolHandlerFunc){
		s.viewfileTool,
		s.editfileTool,
		s.createfileTool,
		s.runbashTool,
	}
	for _, tool := range tools {
		def, handle := tool()
		temp, err := shared.ConvertToMcpTool(def)
		if err != nil {
			return nil, err
		} else {
			s.mcpServer.AddTool(temp, handle)
		}
	}
	return s, nil
}

func (s *Server) Run() error {
	err := server.ServeStdio(s.mcpServer)
	if err != nil {
		return err
	}
	return nil
}
