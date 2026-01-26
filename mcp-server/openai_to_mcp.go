package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func OpenAIFunctionToMCPTool(fn *openai.FunctionDefinition) mcp.Tool {
	inputSchema := convertParameters(fn.Parameters)

	return mcp.NewTool(
		fn.Name,
		mcp.WithDescription(fn.Description),
		mcp.WithRawInputSchema(inputSchema),
	)
}

func convertParameters(params any) json.RawMessage {
	if params == nil {
		return []byte(`{"type": "object", "properties": {}}`)
	}

	switch p := params.(type) {
	case jsonschema.Definition:
		return convertDefinition(p)
	case map[string]any:
		return convertMapSchema(p)
	case []byte:
		return p
	default:
		return marshalJSON(p)
	}
}

func convertDefinition(def jsonschema.Definition) []byte {
	schema := make(map[string]any)

	setString(schema, "type", string(def.Type))
	setString(schema, "description", def.Description)
	setArray(schema, "enum", def.Enum)
	setArray(schema, "required", def.Required)

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
		result[key] = convertDefinitionToMap(def)
	}
	return result
}

func convertDefinitionToMap(def jsonschema.Definition) map[string]any {
	schema := make(map[string]any)

	setString(schema, "type", string(def.Type))
	setString(schema, "description", def.Description)
	setArray(schema, "enum", def.Enum)

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

	return schema
}

func convertDefs(defs map[string]jsonschema.Definition) map[string]any {
	result := make(map[string]any)
	for key, def := range defs {
		result[key] = json.RawMessage(convertDefinition(def))
	}
	return result
}

func convertMapSchema(m map[string]any) []byte {
	schema := make(map[string]any)

	if t, ok := m["type"].(string); ok {
		schema["type"] = t
	}
	if d, ok := m["description"].(string); ok {
		schema["description"] = d
	}
	if e := m["enum"]; e != nil {
		schema["enum"] = e
	}
	if r := m["required"]; r != nil {
		if reqSlice, ok := r.([]any); ok {
			reqStrings := make([]string, len(reqSlice))
			for i, v := range reqSlice {
				if s, ok := v.(string); ok {
					reqStrings[i] = s
				}
			}
			schema["required"] = reqStrings
		}
	}
	if p := m["properties"]; p != nil {
		schema["properties"] = p
	}
	if i := m["items"]; i != nil {
		schema["items"] = i
	}
	if ap := m["additionalProperties"]; ap != nil {
		schema["additionalProperties"] = ap
	}

	data, _ := json.Marshal(schema)
	return data
}

func setString(m map[string]any, key, value string) {
	if value != "" {
		m[key] = value
	}
}

func setArray(m map[string]any, key string, value []string) {
	if len(value) > 0 {
		m[key] = value
	}
}

func marshalJSON(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		return []byte(`{"type": "object"}`)
	}
	return data
}

func ConvertOpenAIToolsToMCP(openaiTools []openai.Tool) []mcp.Tool {
	mcpTools := make([]mcp.Tool, len(openaiTools))

	for i, tool := range openaiTools {
		if tool.Function != nil {
			mcpTools[i] = OpenAIFunctionToMCPTool(tool.Function)
		}
	}

	return mcpTools
}

type CalculatorParams struct {
	Operation string  `json:"operation" description:"Operation to perform" enum:"add,subtract,multiply,divide"`
	A         float64 `json:"a" description:"First number"`
	B         float64 `json:"b" description:"Second number"`
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
	fmt.Printf("MCP Tool: %s - %s\n", mcpTool.Name, mcpTool.Description)

	schema, _ := jsonschema.GenerateSchemaForType(CalculatorParams{})
	autoTool := openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "calculator",
			Description: "Perform arithmetic operations",
			Parameters:  *schema,
		},
	}

	mcpAutoTool := OpenAIFunctionToMCPTool(autoTool.Function)
	_ = mcpAutoTool

	mcpTools := ConvertOpenAIToolsToMCP([]openai.Tool{weatherTool, autoTool})
	fmt.Printf("Converted %d tools\n", len(mcpTools))
}
