package mcpclient

import (
	"context"
	"errors"
	"fmt"
	"multi-agent/service"
	"multi-agent/shared"
	"strings"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

type ClientMgr struct {
	clientMap map[string]*client.Client
}

func NewclientMgr() *ClientMgr {
	return &ClientMgr{
		clientMap: map[string]*client.Client{},
	}
}

func (mgr *ClientMgr) CloseByName(name string) error {
	client, exist := mgr.clientMap[name]
	if !exist {
		return fmt.Errorf("client %s not exist", name)
	}
	err := client.Close()
	if err != nil {
		return err
	}
	delete(mgr.clientMap, name)
	return nil
}

func (mgr *ClientMgr) Close() error {
	var errList []error = nil
	for _, client := range mgr.clientMap {
		err := client.Close()
		if err != nil {
			errList = append(errList, err)
		}
	}
	return errors.Join(errList...)
}

func (mgr *ClientMgr) NewMCPClient(command string, env []string, args ...string) error {
	c, err := client.NewStdioMCPClient(command, env, args...)
	if err != nil {
		return err
	}
	res, err := c.Initialize(context.Background(), mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "sampling-example-client",
				Version: "1.0.0",
			},
			Capabilities: mcp.ClientCapabilities{},
		},
	})
	if err != nil {
		return err
	}
	_, exist := mgr.clientMap[res.ServerInfo.Name]
	if exist {
		return fmt.Errorf("mcp server %s already exist", res.ServerInfo.Name)
	}
	mgr.clientMap[res.ServerInfo.Name] = c
	return nil
}

func (mgr *ClientMgr) LoadAllTools() ([]service.ToolEndPoint, error) {
	var endpoint []service.ToolEndPoint
	var errorList []error
	for _, c := range mgr.clientMap {
		res, err := mgr.loadTools(c)
		if err != nil {
			errorList = append(errorList, err)
		} else {
			endpoint = append(endpoint, res...)
		}
	}
	err := errors.Join(errorList...)
	if err != nil {
		return nil, err
	}
	return endpoint, nil
}

func (mgr *ClientMgr) loadTools(c *client.Client) ([]service.ToolEndPoint, error) {
	res, err := c.ListTools(context.Background(), mcp.ListToolsRequest{})
	if err != nil {
		return nil, err
	}
	endpointList := []service.ToolEndPoint{}
	for _, tool := range res.Tools {
		endpoint := service.ToolEndPoint{
			Name: tool.Name,
			Def:  shared.ConvertToFunctionDefinition(tool),
			Handler: func(args string) (string, error) {
				res, err := c.CallTool(context.Background(), mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      tool.Name,
						Arguments: args,
					},
				})
				if err != nil {
					return "", err
				}
				var builder strings.Builder
				for _, content := range res.Content {
					text, ok := content.(mcp.TextContent)
					if ok {
						builder.WriteString(text.Text)
						builder.WriteByte('\n')
					}
				}
				return builder.String(), nil
			},
		}
		endpointList = append(endpointList, endpoint)
	}
	return endpointList, nil
}
