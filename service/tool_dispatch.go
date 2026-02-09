package service

import (
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
)

type ToolEndPoint struct {
	Name    string
	Def     openai.FunctionDefinition
	Handler func(args string) (string, error)
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
	content := ""
	var err error
	if exist {
		content, err = endpoint.Handler(toolCall.Function.Arguments)
	} else {
		err = fmt.Errorf("Run tool call failed, Can not find tool with name %s", toolCall.Function.Name)
	}
	var builder strings.Builder
	builder.WriteString("### Metadata\n\n")
	builder.WriteString("### Tool Result\n\n")
	builder.WriteString("** Status **\n")
	if err != nil {
		builder.WriteString(fmt.Sprintf("Execute tool call failed, error: %s\n", err))
	} else {
		builder.WriteString("Execute tool call success\n")
		builder.WriteString("** Result **\n")
		builder.WriteString(content)
	}
	res.Content = builder.String()
	return res
}

func (td *ToolDispatcher) GetTools() []openai.Tool {
	res := make([]openai.Tool, 0, len(td.toolMap))
	for _, endpoint := range td.toolMap {
		res = append(res, openai.Tool{
			Type:     openai.ToolTypeFunction,
			Function: &endpoint.Def,
		})
	}
	return res
}
