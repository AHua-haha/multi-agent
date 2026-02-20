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

type OutputFunc func(msg openai.ChatCompletionMessage) bool

type BaseAgent struct {
	actionStack  []openai.ChatCompletionMessage
	input        []openai.ChatCompletionMessage
	toolDispatch *service.ToolDispatcher
}

func NewBaseAgent(instruct string, userInput string, tools *service.ToolDispatcher) *BaseAgent {
	input := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: instruct},
		{Role: openai.ChatMessageRoleUser, Content: userInput},
	}
	return &BaseAgent{
		input:        input,
		toolDispatch: tools,
	}
}

func (a *BaseAgent) chat(client *openai.Client, model string) (*openai.ChatCompletionChoice, error) {
	msgs := []openai.ChatCompletionMessage{}
	msgs = append(msgs, a.input...)
	msgs = append(msgs, a.actionStack...)
	req := openai.ChatCompletionRequest{
		Model:    model,
		Messages: msgs,
		Tools:    a.toolDispatch.GetTools(),
	}
	response, err := client.CreateChatCompletion(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return &response.Choices[0], err
}

func (a *BaseAgent) handleToolCall(toolCalls []openai.ToolCall) {
	for _, call := range toolCalls {
		res := a.toolDispatch.Run(call)
		a.actionStack = append(a.actionStack, res)
		writer.WriteString("<TOOL CALL>\n")
		writer.WriteString(fmt.Sprintf("tool name: %s\ntool args: %s\n", call.Function.Name, call.Function.Arguments))
		writer.WriteString(fmt.Sprintf("result:\n%s", res.Content))
		writer.WriteString("</TOOL CALL>\n")
	}
}
func (a *BaseAgent) Run(client *openai.Client, model string, outputFunc OutputFunc) error {
	f, err := os.OpenFile("agent_log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	writer = bufio.NewWriter(f)

	writer.WriteString(fmt.Sprintf("SYSTEM PROMPT: %s\n", a.input[0].Content))
	writer.WriteString(fmt.Sprintf("User Input: %s\n\n", a.input[1].Content))

	a.actionStack = nil
	for {
		resp, err := a.chat(client, model)
		if err != nil {
			log.Error().Err(err).Msg("chat failed")
			return err
		}
		a.actionStack = append(a.actionStack, resp.Message)
		writer.WriteString("<MSG> clean thinking\n")
		writer.WriteString(fmt.Sprintf("Role: %s\n", resp.Message.Role))
		writer.WriteString(fmt.Sprintf("content:\n%s\n", resp.Message.Content))
		writer.WriteString("</MSG>\n")

		a.handleToolCall(resp.Message.ToolCalls)

		if outputFunc != nil {
			finished := outputFunc(resp.Message)
			if finished {
				break
			}
		}
		if resp.FinishReason == openai.FinishReasonStop {
			break
		}
	}
	writer.Flush()
	return nil
}

var writer *bufio.Writer
