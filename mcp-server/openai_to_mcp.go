package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func OpenAIFunctionToMCPTool(fn *openai.FunctionDefinition) mcp.Tool {
	inputSchema := convertDefinitionToSchema(fn.Parameters)

	return mcp.NewTool(
		fn.Name,
		mcp.WithDescription(fn.Description),
		mcp.WithRawInputSchema(inputSchema),
	)
}

func convertDefinitionToSchema(params any) json.RawMessage {
	if params == nil {
		return json.RawMessage(`{"type": "object", "properties": {}}`)
	}

	switch p := params.(type) {
	case jsonschema.Definition:
		return convertDefinition(p)
	case map[string]any:
		return convertMapToJSONSchema(p)
	case []byte:
		return p
	default:
		return marshalToJSON(p)
	}
}

func convertDefinition(def jsonschema.Definition) json.RawMessage {
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
	if def.Properties != nil {
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
	if def.Defs != nil {
		schema["$defs"] = convertDefs(def.Defs)
	}

	data, _ := json.Marshal(schema)
	return data
}

func convertProperties(props map[string]jsonschema.Definition) map[string]any {
	result := make(map[string]any)
	for key, def := range props {
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
		if def.Properties != nil {
			schema["properties"] = convertProperties(def.Properties)
		}
		if def.Items != nil {
			schema["items"] = convertDefinition(*def.Items)
		}
		if def.AdditionalProperties != nil {
			schema["additionalProperties"] = def.AdditionalProperties
		}
		result[key] = schema
	}
	return result
}

func convertDefs(defs map[string]jsonschema.Definition) map[string]any {
	result := make(map[string]any)
	for key, def := range defs {
		result[key] = convertDefinitionToSchema(def)
	}
	return result
}

func convertMapToJSONSchema(m map[string]any) json.RawMessage {
	schema := make(map[string]any)

	if typeStr, ok := m["type"].(string); ok {
		schema["type"] = typeStr
	}
	if desc, ok := m["description"].(string); ok {
		schema["description"] = desc
	}
	if enums := m["enum"]; enums != nil {
		schema["enum"] = enums
	}
	if required, ok := m["required"].([]any); ok {
		reqStrings := make([]string, len(required))
		for i, r := range required {
			if s, ok := r.(string); ok {
				reqStrings[i] = s
			}
		}
		schema["required"] = reqStrings
	}
	if properties, ok := m["properties"].(map[string]any); ok {
		schema["properties"] = properties
	}
	if items := m["items"]; items != nil {
		schema["items"] = items
	}
	if additionalProperties, ok := m["additionalProperties"].(bool); ok {
		schema["additionalProperties"] = additionalProperties
	}

	data, _ := json.Marshal(schema)
	return data
}

func marshalToJSON(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`{"type": "object"}`)
	}
	return data
}

func ConvertOpenAIToolsToMCP(openaiTools []openai.Tool) []mcp.Tool {
	mcpTools := make([]mcp.Tool, 0, len(openaiTools))

	for _, tool := range openaiTools {
		if tool.Function != nil {
			mcpTools = append(mcpTools, OpenAIFunctionToMCPTool(tool.Function))
		}
	}

	return mcpTools
}

type CalculatorParams struct {
	Operation string  `json:"operation"`
	A         float64 `json:"a"`
	B         float64 `json:"b"`
}

type WeatherParams struct {
	Location string `json:"location" description:"City name"`
	Unit     string `json:"unit,omitempty" description:"Temperature unit" enum:"celsius,fahrenheit"`
}

func UsageExample() {
	weatherTool := openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "get_current_weather",
			Description: "Get the current weather in a given location",
			Parameters: jsonschema.Definition{
				Type: jsonschema.Object,
				Properties: map[string]jsonschema.Definition{
					"location": {
						Type:        jsonschema.String,
						Description: "The city and state, e.g. San Francisco, CA",
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
	fmt.Printf("MCP Tool from Definition: %+v\n", mcpTool)

	schema, _ := jsonschema.GenerateSchemaForType(WeatherParams{})
	autoSchemaTool := openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "get_weather_auto",
			Description: "Get weather using auto-generated schema",
			Parameters:  *schema,
		},
	}

	mcpToolFromAuto := OpenAIFunctionToMCPTool(autoSchemaTool.Function)
	_ = mcpToolFromAuto

	multipleTools := []openai.Tool{weatherTool, autoSchemaTool}
	mcpToolList := ConvertOpenAIToolsToMCP(multipleTools)
	fmt.Printf("Converted %d tools\n", len(mcpToolList))
}
