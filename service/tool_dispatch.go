package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
)

type ToolEndPoint struct {
	Name    string
	Def     openai.FunctionDefinition
	Handler func(args string) (string, error)
}

type ToolExecLog struct {
	ID         int
	ToolCall   openai.ToolCall
	ToolRes    string
	ToolErr    error
}

func (toolLog *ToolExecLog) formatString() string {
	var builder strings.Builder
	builder.WriteString("** Metadata **\n")
	builder.WriteString(fmt.Sprintf("TOOL_LOG_ID: %d\n", toolLog.ID))
	builder.WriteString(fmt.Sprintf("TOOL_NAME: %s\n", toolLog.ToolCall.Function.Name))
	builder.WriteString("** Status **\n")
	if toolLog.ToolErr != nil {
		builder.WriteString(fmt.Sprintf("Execute tool call failed, error: %s\n", toolLog.ToolErr))
	} else {
		builder.WriteString("Execute tool call success\n")
		builder.WriteString("** Result **\n")
		builder.WriteString(toolLog.ToolRes)
	}
	return builder.String()
}

// ReconstructAssistantMessage reconstructs the assistant message for this tool call
func (toolLog *ToolExecLog) ReconstructAssistantMessage() openai.ChatCompletionMessage {
	return openai.ChatCompletionMessage{
		Role: openai.ChatMessageRoleAssistant,
		ToolCalls: []openai.ToolCall{
			toolLog.ToolCall,
		},
	}
}

// ReconstructToolMessage reconstructs the tool response message
func (toolLog *ToolExecLog) ReconstructToolMessage() openai.ChatCompletionMessage {
	content := toolLog.ToolRes
	if toolLog.ToolErr != nil {
		content = fmt.Sprintf("Error: %s", toolLog.ToolErr)
	}

	return openai.ChatCompletionMessage{
		Role:       openai.ChatMessageRoleTool,
		ToolCallID: toolLog.ToolCall.ID,
		Content:    content,
	}
}

type ToolDispatcher struct {
	toolMap map[string]ToolEndPoint
	toolLog []*ToolExecLog
}

func NewToolDispatcher(toolLog []*ToolExecLog) *ToolDispatcher {
	return &ToolDispatcher{
		toolMap: map[string]ToolEndPoint{},
		toolLog: toolLog,
	}
}
func (td *ToolDispatcher) GetToolLog() []*ToolExecLog {
	return td.toolLog
}

func (td *ToolDispatcher) ResetTools() {
	td.toolMap = map[string]ToolEndPoint{}
}
func (td *ToolDispatcher) RegisterToolEndpoint(endpoints ...ToolEndPoint) error {
	err := []error{}
	for _, endpoint := range endpoints {
		_, exist := td.toolMap[endpoint.Name]
		if exist {
			err = append(err, fmt.Errorf("tool with name %s already exist", endpoint.Name))
		} else {
			td.toolMap[endpoint.Name] = endpoint
		}
	}
	return errors.Join(err...)
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
	log := ToolExecLog{
		ID:         len(td.toolLog),
		ToolCall:   toolCall,
		ToolRes:    content,
		ToolErr:    err,
	}
	td.toolLog = append(td.toolLog, &log)
	res.Content = log.formatString()
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

func (td *ToolDispatcher) DebugTools() {
	for _, tool := range td.toolMap {
		fmt.Printf("tool.Name: %v\n", tool.Name)
		data, _ := json.Marshal(tool.Def)
		fmt.Printf("%s\n", string(data))
	}
}
