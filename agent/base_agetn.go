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

func NewBaseAgent(instruct string, userInput string, tools *service.ToolDispatcher, prevToolMessages []openai.ChatCompletionMessage) *BaseAgent {
	// Build input messages with system prompt, user input, and previous tool logs
	for _, msg := range prevToolMessages {
		fmt.Printf("msg.Content: %v\n", msg.Content)
	}
	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: instruct},
		{Role: openai.ChatMessageRoleUser, Content: userInput},
	}

	// Add previous tool messages if provided
	if len(prevToolMessages) > 0 {
		messages = append(messages, prevToolMessages...)
	}

	return &BaseAgent{
		input:        messages,
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
		Writer.WriteString("<TOOL CALL>\n")
		Writer.WriteString(fmt.Sprintf("tool name: %s\ntool args: %s\n", call.Function.Name, call.Function.Arguments))
		Writer.WriteString(fmt.Sprintf("result:\n%s", res.Content))
		Writer.WriteString("</TOOL CALL>\n")
	}
}
func (a *BaseAgent) Run(client *openai.Client, model string, outputFunc OutputFunc) error {
	f, err := os.OpenFile("agent_log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	Writer = bufio.NewWriter(f)

	Writer.WriteString(fmt.Sprintf("SYSTEM PROMPT: %s\n", a.input[0].Content))
	Writer.WriteString(fmt.Sprintf("%s\n\n", a.input[1].Content))

	a.actionStack = nil
	for {
		resp, err := a.chat(client, model)
		if err != nil {
			log.Error().Err(err).Msg("chat failed")
			return err
		}
		a.actionStack = append(a.actionStack, resp.Message)
		Writer.WriteString("<MSG> clean thinking\n")
		Writer.WriteString(fmt.Sprintf("Role: %s\n", resp.Message.Role))
		Writer.WriteString(fmt.Sprintf("content:\n%s\n", resp.Message.Content))
		Writer.WriteString("</MSG>\n")

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
	Writer.Flush()
	return nil
}

var Writer *bufio.Writer
