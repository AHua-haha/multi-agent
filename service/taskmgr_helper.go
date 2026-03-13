package service

import (
	"github.com/sashabaranov/go-openai"
)

// GetAllTaskToolCallMessages returns all tool calls from all previous tasks as OpenAI messages
func (mgr *TaskMgr) GetAllTaskToolCallMessages() []openai.ChatCompletionMessage {
	var messages []openai.ChatCompletionMessage

	if len(mgr.PreTasks) == 0 {
		return messages
	}

	// Iterate through all completed tasks
	for _, task := range mgr.PreTasks {
		switch t := task.(type) {
		case *ExploreTask:
			for _, ctx := range t.Context {
				if ctx.ToolLog != nil {
					messages = append(messages, ctx.ToolLog.ReconstructAssistantMessage())
					messages = append(messages, ctx.ToolLog.ReconstructToolMessage())
				}
			}
		case *ReasonTask:
			// Reason tasks typically don't have context items
		case *BuildTask:
			for _, ctx := range t.Context {
				if ctx.ToolLog != nil {
					messages = append(messages, ctx.ToolLog.ReconstructAssistantMessage())
					messages = append(messages, ctx.ToolLog.ReconstructToolMessage())
				}
			}
		case *VerifyTask:
			for _, ctx := range t.Context {
				if ctx.ToolLog != nil {
					messages = append(messages, ctx.ToolLog.ReconstructAssistantMessage())
					messages = append(messages, ctx.ToolLog.ReconstructToolMessage())
				}
			}
		}
	}

	return messages
}