package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/sashabaranov/go-openai"
)

func TestReactAgent_chat(t *testing.T) {
	t.Run("test minimax api", func(t *testing.T) {
		config := openai.DefaultConfig("sk-cp-txjkExo9UcNpcAIoKuw0OMjOOcSEygZ7OpGDBaY0QQHwUJH_Y_R73dmv3tTuEedPLquZyMkMX9EI50nz7cYESoDgh21B-64M1Ezf76mToHKv_NTFVUaz6Xo")
		config.BaseURL = "https://api.minimaxi.com/v1"
		client := openai.NewClientWithConfig(config)
		fmt.Print("start client chat\n")

		// 3. Create the request
		// Use MiniMax models like "MiniMax-M2.1" or "abab6.5-chat"
		resp, err := client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model: "MiniMax-M2.1",
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleUser,
						Content: "Write a short Go function to calculate Fibonacci numbers.",
					},
				},
			},
		)
		if err != nil {
			fmt.Printf("err: %v\n", err)
		}
		fmt.Printf("resp: %v\n", resp)
	})
}
