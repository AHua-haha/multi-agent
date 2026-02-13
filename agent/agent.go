package agent

import (
	"bufio"
	"context"
	"fmt"
	"multi-agent/service"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/sashabaranov/go-openai"
)

type ReactAgent struct {
	actionStack []openai.ChatCompletionMessage
	taskMgr     *service.TaskMgr

	toolDispatch *service.ToolDispatcher
	client       *openai.Client
}

func NewAgent(client *openai.Client) *ReactAgent {
	agent := &ReactAgent{
		client:       client,
		toolDispatch: service.NewToolDispatcher(),
		taskMgr:      &service.TaskMgr{},
	}
	agent.toolDispatch.RegisterToolEndpoint(agent.taskMgr.CreateTaskTool())
	agent.toolDispatch.RegisterToolEndpoint(agent.taskMgr.FinishTaskTool())
	return agent
}
func (a *ReactAgent) AddTools(endpoints []service.ToolEndPoint) error {
	err := a.toolDispatch.RegisterToolEndpoint(endpoints...)
	if err != nil {
		return err
	}
	return nil
}

var systemPrompt = `
You are an expert Coding Agent. Your goal is to solve technical tasks with precision, minimal verbosity, and strict tool discipline.
When you decide to call a tool, output the tool call immediately.
Do not add "I will now call...", "Searching for...", or any other text before, during, or after a tool call. The output for that turn must contain ONLY the tool call block

## Communication Style
** Short & Concise **: Provide direct answers. Eliminate pleasantries ("Certainly!", "I hope this helps").
** Technical Density **: Use technical terms accurately but briefly. If a code snippet explains the solution, prioritize the code over text.
`

func (a *ReactAgent) chat() (*openai.ChatCompletionChoice, error) {
	msgs := []openai.ChatCompletionMessage{}
	msgs = append(msgs, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: systemPrompt,
	})
	msgs = append(msgs, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: a.taskMgr.GetTaskContextPrompt(),
	})
	msgs = append(msgs, a.actionStack...)
	if a.taskMgr.CurrentTask == nil {
		msgs = append(msgs, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: service.CreateTaskInstruct,
		})
	} else {
		msgs = append(msgs, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: service.DoingTaskInstruct,
		})
	}
	req := openai.ChatCompletionRequest{
		Model:    "MiniMax-M2.5",
		Messages: msgs,
		Tools:    a.toolDispatch.GetTools(),
	}
	response, err := a.client.CreateChatCompletion(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return &response.Choices[0], err
}

func (a *ReactAgent) handleToolCall(toolCalls []openai.ToolCall) {
	for _, call := range toolCalls {
		res := a.toolDispatch.Run(call)
		writer.WriteString("<TOOL CALL>\n")
		writer.WriteString(fmt.Sprintf("tool name: %s\ntool args: %s\n", call.Function.Name, call.Function.Arguments))
		writer.WriteString(fmt.Sprintf("result:\n%s", res.Content))
		writer.WriteString("</TOOL CALL>\n")
		a.actionStack = append(a.actionStack, res)
	}
	if toolCalls[len(toolCalls)-1].Function.Name == "create_task" {
		writer.WriteString("<TASK INFO>\n")
		writer.WriteString(a.taskMgr.GetTaskContextPrompt())
		writer.WriteString("<TASK INFO>\n")
	}
	if toolCalls[len(toolCalls)-1].Function.Name == "finish_task" {
		a.actionStack = nil
		a.toolDispatch.ResetLog()
		writer.WriteString("<TASK INFO>\n")
		writer.WriteString(a.taskMgr.GetTaskContextPrompt())
		writer.WriteString("<TASK INFO>\n")
	}
}

var writer *bufio.Writer

func (a *ReactAgent) run(input string) {
	a.actionStack = nil
	a.taskMgr.Reset(input)

	f, err := os.OpenFile("agent_log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	writer = bufio.NewWriter(f)

	writer.WriteString(fmt.Sprintf("User Input: %s\n\n", input))
	writer.WriteString(fmt.Sprintf("SYSTEM PROMPT: %s\n", systemPrompt))
	for {
		resp, err := a.chat()
		if err != nil {
			log.Error().Err(err).Msg("chat failed")
			break
		}
		if resp.FinishReason == openai.FinishReasonStop {
			writer.WriteString("FINAL RESP:\n")
			writer.WriteString(fmt.Sprintf("Role: %s\n", resp.Message.Role))
			writer.WriteString(fmt.Sprintf("content:\n%s\n", resp.Message.Content))
			break
		}
		writer.WriteString("<MSG> clean thinking\n")
		writer.WriteString(fmt.Sprintf("Role: %s\n", resp.Message.Role))
		writer.WriteString(fmt.Sprintf("content:\n%s\n", resp.Message.Content))
		writer.WriteString("</MSG>\n")
		resp.Message.Content = ""
		a.actionStack = append(a.actionStack, resp.Message)
		a.handleToolCall(resp.Message.ToolCalls)
		writer.Flush()
	}
}
