package mcpclient_test

import (
	"encoding/json"
	"fmt"
	mcpclient "multi-agent/mcp-client"
	"testing"
)

func TestClientMgr(t *testing.T) {
	t.Run("test load tools", func(t *testing.T) {
		mgr := mcpclient.NewclientMgr()
		mgr.NewMCPClient("uv", nil, "run", "/root/multi-agent/cmds/lsp-mcp.py")
		mgr.NewMCPClient("/root/multi-agent/bin/mcpserver", nil)
		res, err := mgr.LoadAllTools()
		if err != nil {
			fmt.Printf("err: %v\n", err)
		}
		for _, endpoint := range res {
			bytes, _ := json.MarshalIndent(endpoint.Def, "", "  ")
			fmt.Printf("%s\n", string(bytes))
		}
	})
}
