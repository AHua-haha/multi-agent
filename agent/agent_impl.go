package agent

import (
	"multi-agent/service"

	"github.com/sashabaranov/go-openai"
)

func (w *Workflow) OrchestratorAgent() (string, error) {
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
	tools := service.NewToolDispatcher(w.toolLog)
	tools.RegisterToolEndpoint(w.taskMgr.CreateTaskTool())
	userInput := w.taskMgr.GetTaskContextPrompt()
	agent := NewBaseAgent(instruct, userInput, tools)

	var final_msg string = ""

	outputFunc := func(msg openai.ChatCompletionMessage) bool {
		if len(msg.ToolCalls) == 0 {
			final_msg = msg.Content
		}
		return true
	}

	err := agent.Run(w.client, "glm-5", outputFunc)
	if err != nil {
		return "", err
	}
	w.toolLog = tools.GetToolLog()
	return final_msg, err
}

func (w *Workflow) WorkerAgent() error {
	instruct := `
You are a worker agent, the 'User Primary Goal' are decomposeed to sub tasks. Your job is to accomplish the 'Current Task'.
Focus on the 'Conclusion Requirements' and the 'Background Context Requirements', make best effort to meet these requirements.

IMPORTANT: after you accomplish the current sub task, MUST IMMEDIATELY use the 'finish_task' tool to record the output of this current task.
IMPORTANT: do not continue the 'User Primary Goal' after you call 'finish_task' tool to finish the current task.
IMPORTANT: your goal is not to complete the 'User Primary Goal', instead, Focus on the 'Current Task', your goal is to complete the 'Current Task'.
`
	tools := service.NewToolDispatcher(w.toolLog)
	mcpTool, err := w.mcpclient.LoadAllTools()
	if err != nil {
		return err
	}
	tools.RegisterToolEndpoint(w.taskMgr.FinishTaskTool())
	tools.RegisterToolEndpoint(mcpTool...)
	userInput := w.taskMgr.GetTaskContextPrompt()
	agent := NewBaseAgent(instruct, userInput, tools)

	outputFunc := func(msg openai.ChatCompletionMessage) bool {
		if len(msg.ToolCalls) != 0 {
			for _, call := range msg.ToolCalls {
				if call.Function.Name == "finish_task" {
					return true
				}
			}
		}
		return false
	}

	err = agent.Run(w.client, "glm-5", outputFunc)
	if err != nil {
		return err
	}
	w.toolLog = tools.GetToolLog()
	w.taskMgr.FillToolLog(w.toolLog)
	return nil
}
func (w *Workflow) ContextAgent() error {
	instruct := `
You are the **Context Refine Agent**. Your goal is to refine the context, make the context short and concise, reduce the unnecessary infomation.

You are given the previous 'Task History', each task has the 'Exprected Output', the output has two kinds, the 'Conclusions Requirements:' and 'Background Context Requirements'.
- ** Conclusions Requirements **: These are the direct facts and conclusions to extract as the output of the task.
- ** Background Context Requirements **: background context to observe and record,

Your job is to refine the context output, make the context output short and concise, reduce unnecessary context.
`
	tools := service.NewToolDispatcher(w.toolLog)
	mcpTool, err := w.mcpclient.LoadAllTools()
	if err != nil {
		return err
	}
	tools.RegisterToolEndpoint(w.taskMgr.RefineContextTool())
	tools.RegisterToolEndpoint(mcpTool...)
	userInput := w.taskMgr.GetInputForRefineContext()
	agent := NewBaseAgent(instruct, userInput, tools)

	err = agent.Run(w.client, "glm-5", nil)
	if err != nil {
		return err
	}
	w.toolLog = tools.GetToolLog()
	return nil
}
