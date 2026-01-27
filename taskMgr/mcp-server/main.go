package main

import (
	"multi-agent/taskMgr"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	mgr := &taskMgr.TaskMgr{}
	s := server.NewMCPServer("Task Manager Mcp Server", "v1.0", server.WithToolCapabilities(true))
	mgr.Config(s)
	err := server.ServeStdio(s)
	if err != nil {
		return
	}
}
