package service

import (
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type ExploreTask struct {
	Task         string
	ExpectOutput string
	Context      []ContextItem
}
type ReasonTask struct {
	Task         string
	ExpectOutput string
	Conclusion   string
}
type BuildTask struct {
	Task      string
	ChangeLog string
	Context   []ContextItem
}
type VerifyTask struct {
	Task       string
	Conclusion string
}

func CreateExploreTask() ToolEndPoint {
	def := openai.FunctionDefinition{
		Name:        "create_explore_task",
		Description: "Creates an exploration task to investigate and gather information about the codebase or system",
		Parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"Task": {
					Type:        jsonschema.String,
					Description: "The specific exploration task to perform, e.g., 'Find all HTTP handlers in the codebase'",
				},
				"ExpectOutput": {
					Type:        jsonschema.String,
					Description: "What kind of output is expected from this exploration task, e.g., 'List of handler functions with their routes'",
				},
			},
			Required: []string{"Task", "ExpectOutput"},
		},
	}
	endpoint := ToolEndPoint{
		Name: "create_explore_task",
		Def:  def,
	}
	return endpoint
}
