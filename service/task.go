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

func CreateReasonTask() ToolEndPoint {
	def := openai.FunctionDefinition{
		Name:        "create_reason_task",
		Description: "Creates a reasoning task to analyze information and draw conclusions based on gathered data",
		Parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"Task": {
					Type:        jsonschema.String,
					Description: "The reasoning task to perform, e.g., 'Analyze the architecture pattern used in the project'",
				},
				"ExpectOutput": {
					Type:        jsonschema.String,
					Description: "What kind of reasoning output is expected, e.g., 'Detailed analysis with supporting evidence'",
				},
			},
			Required: []string{"Task", "ExpectOutput"},
		},
	}
	endpoint := ToolEndPoint{
		Name: "create_reason_task",
		Def:  def,
	}
	return endpoint
}

func CreateBuildTask() ToolEndPoint {
	def := openai.FunctionDefinition{
		Name:        "create_build_task",
		Description: "Creates a build task to make modifications or additions to the codebase",
		Parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"Task": {
					Type:        jsonschema.String,
					Description: "The build task to perform, e.g., 'Implement error handling in the API handler'",
				},
			},
			Required: []string{"Task"},
		},
	}
	endpoint := ToolEndPoint{
		Name: "create_build_task",
		Def:  def,
	}
	return endpoint
}

func CreateVerifyTask() ToolEndPoint {
	def := openai.FunctionDefinition{
		Name:        "create_verify_task",
		Description: "Creates a verification task to test and validate changes or implementations",
		Parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"Task": {
					Type:        jsonschema.String,
					Description: "The verification task to perform, e.g., 'Test the error handling implementation'",
				},
			},
			Required: []string{"Task"},
		},
	}
	endpoint := ToolEndPoint{
		Name: "create_verify_task",
		Def:  def,
	}
	return endpoint
}
