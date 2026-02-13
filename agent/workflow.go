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

func (w *Workflow) OrchestratorAgent() error {
	instruct := `
You are the **Task Orchestrator** agent. You decompose 'User Primary Goal' into the smallest possible units of task.
Analyze the 'Task History' against the 'User Primary Goal'. You must choose one of two paths:

#### PATH A: GOAL NOT COMPLETED (DECOMPOSE & CREATE TASK)
If information is missing or a multi-step process is still underway:
1. **Decompose**: Identify the immediate next logical task.
2. **Create Atomic Task**: Define a single, focused task, the task should have only one goal, the task MUST be atomic, NEVER create too complex task with multiple goals.
3. **Structured Expected Output Requirements**: Define the expected "Primary Conclusions" and "Background Context" for the task.
   - NEVER define too complex 'Expected Output', the 'Expected Output' should be simple and focused, 2-3 most essential items is the best.
   - ** Conclusions Requirements **: These are the direct facts and conclusions to extract as the output of the task.
   - ** Background Context Requirements **: background context to observe and record,

#### PATH B: GOAL COMPLETED (FINALIZE)
If all the 'Task History' provide a full answer:
1. **Synthesize**: Combine all facts into a coherent response.
2. **Nuance**: Add relevant "Background Context" to provide helpful background or warnings.
3. **Finalize**: Deliver the final response to the user directly.

IMPORTANT: never assign a task that is too complex and have multiple goals, decompose complex goals into the smallest possible units of task.
IMPORTANT: The 'Eexpected Ooutput Requirements' must be highly focused. Do not ask for "everything." Ask for the 1-2 most critical facts.
`
	tools := service.NewToolDispatcher()
	tools.RegisterToolEndpoint(w.taskMgr.CreateTaskTool())
	userInput := w.taskMgr.GetTaskContextPrompt()
	agent := NewBaseAgent(instruct, userInput, tools)
	err := agent.Run(w.client, "MiniMax-M2.5")
	if err != nil {
		return err
	}
	return nil
}

func (w *Workflow) WorkerAgent() error {
	instruct := `
You are a worker agent, the 'User Primary Goal' are decomposeed to sub tasks. Your job is to accomplish the 'Current Task'.
Focus on the 'Conclusion Requirements' and the 'Background Context Requirements', make best effort to meet these requirements.

IMPORTANT: after you accomplish the current sub task, MUST IMMEDIATELY use the 'finish_task' tool to record the output of this current task.
IMPORTANT: do not continue the 'User Primary Goal' after you call 'finish_task' tool to finish the current task.
`
	tools := service.NewToolDispatcher()
	mcpTool, err := w.mcpclient.LoadAllTools()
	if err != nil {
		return err
	}
	tools.RegisterToolEndpoint(w.taskMgr.FinishTaskTool())
	tools.RegisterToolEndpoint(mcpTool...)
	userInput := w.taskMgr.GetTaskContextPrompt()
	agent := NewBaseAgent(instruct, userInput, tools)
	err = agent.Run(w.client, "MiniMax-M2.5")
	if err != nil {
		return err
	}
	return nil
}

func (w *Workflow) SingleAgent() error {
	w.taskMgr = &service.TaskMgr{}
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter text: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		w.taskMgr.Reset(input)

		log.Info().Msg("agent start running")
		for {
			err := w.OrchestratorAgent()
			if err != nil {
				log.Error().Err(err).Msg("run orchestrator agent failed")
				break
			}
			err = w.WorkerAgent()
			if err != nil {
				log.Error().Err(err).Msg("run worker agent failed")
				break
			}
		}
		log.Info().Msg("agent finish running")
	}
}
