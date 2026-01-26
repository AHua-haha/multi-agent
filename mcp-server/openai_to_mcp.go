package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func OpenAIFunctionToMCPTool(fn *openai.FunctionDefinition) mcp.Tool {
	inputSchema := []byte(`{"type": "object", "properties": {}}`)
	if def, ok := fn.Parameters.(jsonschema.Definition); ok {
		inputSchema, _ = json.Marshal(def)
	}

	return mcp.NewTool(
		fn.Name,
		mcp.WithDescription(fn.Description),
		mcp.WithRawInputSchema(inputSchema),
	)
}

func ConvertOpenAIToolsToMCP(tools []openai.Tool) []mcp.Tool {
	result := make([]mcp.Tool, len(tools))
	for i, tool := range tools {
		if tool.Function != nil {
			result[i] = OpenAIFunctionToMCPTool(tool.Function)
		}
	}
	return result
}

type CalculatorParams struct {
	Operation string  `json:"operation" description:"Operation" enum:"add,subtract,multiply,divide"`
	A         float64 `json:"a" description:"First number"`
	B         float64 `json:"b" description:"Second number"`
}

func UsageExample() {
	weatherTool := openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "get_current_weather",
			Description: "Get weather for a location",
			Parameters: jsonschema.Definition{
				Type: jsonschema.Object,
				Properties: map[string]jsonschema.Definition{
					"location": {
						Type:        jsonschema.String,
						Description: "City name",
					},
					"unit": {
						Type: jsonschema.String,
						Enum: []string{"celsius", "fahrenheit"},
					},
				},
				Required: []string{"location"},
			},
		},
	}

	mcpTool := OpenAIFunctionToMCPTool(weatherTool.Function)
	fmt.Println(mcpTool.Name, mcpTool.Description)

	schema, _ := jsonschema.GenerateSchemaForType(CalculatorParams{})
	autoTool := openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "calculator",
			Description: "Calculate numbers",
			Parameters:  *schema,
		},
	}

	mcpAutoTool := OpenAIFunctionToMCPTool(autoTool.Function)
	_ = mcpAutoTool

	tools := ConvertOpenAIToolsToMCP([]openai.Tool{weatherTool, autoTool})
	fmt.Printf("Converted %d tools\n", len(tools))
}
