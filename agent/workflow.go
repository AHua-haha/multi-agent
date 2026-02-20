package agent

import (
	"bufio"
	"fmt"
	mcpclient "multi-agent/mcp-client"
	"multi-agent/service"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/sashabaranov/go-openai"
)

type Workflow struct {
	mcpclient *mcpclient.ClientMgr
	client    *openai.Client

	toolLog []*service.ToolExecLog

	taskMgr *service.TaskMgr
}

func (w *Workflow) Close() error {
	err := w.mcpclient.Close()
	if err != nil {
		return err
	}
	return nil
}

func (w *Workflow) Init() error {
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		return fmt.Errorf("api key API_KEY not set")
	}
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = "https://open.bigmodel.cn/api/paas/v4"
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

func (w *Workflow) Run() error {
	w.taskMgr = &service.TaskMgr{}
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter text: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		w.taskMgr.Reset(input)

		log.Info().Msg("agent start running")
		for {
			res, err := w.OrchestratorAgent()
			if err != nil {
				log.Error().Err(err).Msg("run orchestrator agent failed")
				break
			}
			if res != "" {
				fmt.Printf("Final Response:\n%s\n", res)
				break
			}
			err = w.WorkerAgent()
			if err != nil {
				log.Error().Err(err).Msg("run worker agent failed")
				break
			}
			w.ContextAgent()
		}
		log.Info().Msg("agent finish running")
	}
}
