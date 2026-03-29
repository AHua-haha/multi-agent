package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	mcpclient "multi-agent/mcp-client"
	"multi-agent/service"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/sashabaranov/go-openai"
)

type Workflow struct {
	mcpclient *mcpclient.ClientMgr
	client    *openai.Client

	toolDispatcher *service.ToolDispatcher

	taskMgr *service.TaskMgr
}

func NewWorkFlow() *Workflow {
	w := &Workflow{
		toolDispatcher: &service.ToolDispatcher{},
	}
	w.taskMgr = &service.TaskMgr{
		ToolDispatcher: w.toolDispatcher,
	}
	return w
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

	// w.mcpclient = mcpclient.NewclientMgr()
	// err := w.mcpclient.NewMCPClient("uv", nil, "run", "/root/multi-agent/cmds/lsp-mcp.py")
	// if err != nil {
	// 	return err
	// }
	// log.Info().Any("server", "lsp-mcp").Msg("create mcp client success")

	// err = w.mcpclient.NewMCPClient("/root/multi-agent/bin/mcpserver", nil)
	// if err != nil {
	// 	return err
	// }
	// log.Info().Any("server", "mcpserver").Msg("create mcp client success")
	return nil
}

func (w *Workflow) Run() error {
	for {
		scanner := bufio.NewScanner(os.Stdin)
		var input struct {
			Task string
		}
		if !scanner.Scan() {
			return fmt.Errorf("can not read from stdin")
		}
		line := scanner.Text() // Automatically trims the newline
		err := json.Unmarshal([]byte(line), &input)
		if err != nil {
			log.Error().Err(err).Msg("parse input task failed")
			return err
		}
		w.taskMgr.Reset(input.Task)

		log.Info().Msg("agent start running")
		for {
			res, err := w.OrchestratorAgent()
			if err != nil {
				log.Error().Err(err).Msg("run orchestrator agent failed")
				break
			}
			if res != "" {
				println("DONE")
				// fmt.Printf("Final Response:\n%s\n", res)
				break
			}
			err = w.WorkerAgent()
			if err != nil {
				log.Error().Err(err).Msg("run worker agent failed")
				break
			}
			// w.ContextAgent()
		}
		log.Info().Msg("agent finish running")
	}
}
