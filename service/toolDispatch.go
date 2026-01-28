package service

import (
	"fmt"

	"github.com/sashabaranov/go-openai"
)

type ToolEndPoint struct {
	Name    string
	def     openai.FunctionDefinition
	handler func(args string) string
}

type ToolDispatcher struct {
	toolMap map[string]*ToolEndPoint
}

func (td *ToolDispatcher) Run(toolCall openai.ToolCall) openai.ChatCompletionMessage {
	endpoint, exist := td.toolMap[toolCall.Function.Name]
	res := openai.ChatCompletionMessage{
		Role:       "tool",
		ToolCallID: toolCall.ID,
	}
	if exist {
		res.Content = endpoint.handler(toolCall.Function.Arguments)
	} else {
		res.Content = fmt.Sprintf("Run tool call failed, Can not find tool with name %s", toolCall.Function.Name)
	}
	return res
}

func (td *ToolDispatcher) GetTools() []openai.Tool {
	res := make([]openai.Tool, 0, len(td.toolMap))
	for _, endpoint := range td.toolMap {
		res = append(res, openai.Tool{
			Type:     openai.ToolTypeFunction,
			Function: &endpoint.def,
		})
	}
	return res
}
