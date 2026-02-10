package agent

import (
	"bufio"
	"fmt"
	mcpclient "multi-agent/mcp-client"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/sashabaranov/go-openai"
)

type Workflow struct {
	mcpclient *mcpclient.ClientMgr
	client    *openai.Client
}

func (w *Workflow) Close() error {
	err := w.mcpclient.Close()
	if err != nil {
		return err
	}
	return nil
}

func (w *Workflow) Init() error {
	apiKey := os.Getenv("MINIMAX_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("api key MINIMAX_API_KEY not set")
	}
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = "https://api.minimaxi.com/v1"
	w.client = openai.NewClientWithConfig(config)
	log.Info().Msg("create openai client success")

	w.mcpclient = mcpclient.NewclientMgr()
	err := w.mcpclient.NewMCPClient("uv", nil, "run", "/root/multi-agent/cmds/lsp-mcp.py")
	if err != nil {
		return err
	}
	log.Info().Any("server", "lsp-mcp").Msg("create mcp client success")

	err = w.mcpclient.NewMCPClient("/root/multi-agent/bin/mcpserver", nil)
	if err != nil {
		return err
	}
	log.Info().Any("server", "mcpserver").Msg("create mcp client success")
	return nil
}

func (w *Workflow) SingleAgent() error {
	agent := NewAgent(w.client)
	endpoints, err := w.mcpclient.LoadAllTools()
	if err != nil {
		log.Error().Err(err).Msg("load all mcp tools failed")
		return err
	}
	agent.AddTools(endpoints)
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter text: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		log.Info().Msg("agent start running")
		agent.run(input)
		log.Info().Msg("agent finish running")
	}
}
