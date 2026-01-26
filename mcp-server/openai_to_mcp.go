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
		inputSchema = convertDefinition(def)
	}

	return mcp.NewTool(
		fn.Name,
		mcp.WithDescription(fn.Description),
		mcp.WithRawInputSchema(inputSchema),
	)
}

func convertDefinition(def jsonschema.Definition) []byte {
	schema := make(map[string]any)

	if def.Type != "" {
		schema["type"] = string(def.Type)
	}
	if def.Description != "" {
		schema["description"] = def.Description
	}
	if len(def.Enum) > 0 {
		schema["enum"] = def.Enum
	}
	if len(def.Required) > 0 {
		schema["required"] = def.Required
	}
	if len(def.Properties) > 0 {
		schema["properties"] = convertProperties(def.Properties)
	}
	if def.Items != nil {
		schema["items"] = convertDefinition(*def.Items)
	}
	if def.AdditionalProperties != nil {
		schema["additionalProperties"] = def.AdditionalProperties
	}
	if def.Nullable {
		schema["nullable"] = true
	}
	if def.Ref != "" {
		schema["$ref"] = def.Ref
	}
	if len(def.Defs) > 0 {
		schema["$defs"] = convertDefs(def.Defs)
	}

	data, _ := json.Marshal(schema)
	return data
}

func convertProperties(props map[string]jsonschema.Definition) map[string]any {
	result := make(map[string]any, len(props))
	for key, def := range props {
		result[key] = definitionToMap(def)
	}
	return result
}

func definitionToMap(def jsonschema.Definition) map[string]any {
	prop := make(map[string]any)

	if def.Type != "" {
		prop["type"] = string(def.Type)
	}
	if def.Description != "" {
		prop["description"] = def.Description
	}
	if len(def.Enum) > 0 {
		prop["enum"] = def.Enum
	}
	if len(def.Properties) > 0 {
		prop["properties"] = convertProperties(def.Properties)
	}
	if def.Items != nil {
		prop["items"] = convertDefinition(*def.Items)
	}
	if def.AdditionalProperties != nil {
		prop["additionalProperties"] = def.AdditionalProperties
	}
	if def.Nullable {
		prop["nullable"] = true
	}
	if def.Ref != "" {
		prop["$ref"] = def.Ref
	}

	return prop
}

func convertDefs(defs map[string]jsonschema.Definition) map[string]any {
	result := make(map[string]any, len(defs))
	for key, def := range defs {
		result[key] = json.RawMessage(convertDefinition(def))
	}
	return result
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
