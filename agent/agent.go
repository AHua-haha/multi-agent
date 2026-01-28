package agent

import (
	"context"
	"multi-agent/service"

	"github.com/sashabaranov/go-openai"
)

type ReactAgent struct {
	msg []openai.ChatCompletionMessage

	toolDispatch *service.ToolDispatcher

	taskMgr *service.TaskMgr

	client *openai.Client
}

func (a *ReactAgent) chat() (*openai.ChatCompletionChoice, error) {
	req := openai.ChatCompletionRequest{
		Model:    "",
		Messages: a.msg,
		Tools:    a.toolDispatch.GetTools(),
	}
	response, err := a.client.CreateChatCompletion(context.Background(), req)
	return &response.Choices[0], err
}

func (a *ReactAgent) handleToolCall(toolCalls []openai.ToolCall) {
	for _, call := range toolCalls {
		res := a.toolDispatch.Run(call)
		a.msg = append(a.msg, res)
	}
}

func (a *ReactAgent) workflow() {
	for {
		resp, _ := a.chat()
		if resp.FinishReason == openai.FinishReasonStop {
			break
		}
		a.msg = append(a.msg, resp.Message)
		a.handleToolCall(resp.Message.ToolCalls)
	}
}
