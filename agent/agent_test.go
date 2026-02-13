package agent

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/sashabaranov/go-openai"
)

func TestReactAgent_chat(t *testing.T) {
	t.Run("test minimax api", func(t *testing.T) {
		apiKey := os.Getenv("MINIMAX_API_KEY")
		if apiKey == "" {
			return
		}
		config := openai.DefaultConfig(apiKey)
		config.BaseURL = "https://api.minimaxi.com/v1"
		client := openai.NewClientWithConfig(config)

		// 3. Create the request
		// Use MiniMax models like "MiniMax-M2.1" or "abab6.5-chat"
		resp, err := client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model: "MiniMax-M2.5",
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleUser,
						Content: "tell me your model version and your knowledge cut off time",
					},
				},
			},
		)
		if err != nil {
			fmt.Printf("err: %v\n", err)
		}
		fmt.Printf("resp: %v\n", resp.Choices[0].Message.Content)
	})
}
