package agent

import (
	"context"
	"multi-agent/service"

	"github.com/sashabaranov/go-openai"
)

type ReactAgent struct {
	actionStack []openai.ChatCompletionMessage

	toolDispatch *service.ToolDispatcher

	taskMgr *service.TaskMgr

	client *openai.Client
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
	req := openai.ChatCompletionRequest{
		Model:    "",
		Messages: msgs,
		Tools:    a.toolDispatch.GetTools(),
	}
	response, err := a.client.CreateChatCompletion(context.Background(), req)
	return &response.Choices[0], err
}

func (a *ReactAgent) handleToolCall(toolCalls []openai.ToolCall) {
	for _, call := range toolCalls {
		res := a.toolDispatch.Run(call)
		a.actionStack = append(a.actionStack, res)
	}
}

func (a *ReactAgent) workflow() {
	for {
		resp, _ := a.chat()
		if resp.FinishReason == openai.FinishReasonStop {
			break
		}
		a.actionStack = append(a.actionStack, resp.Message)
		a.handleToolCall(resp.Message.ToolCalls)
	}
}
