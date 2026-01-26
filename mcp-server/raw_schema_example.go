package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

var WeatherTool = mcplib.NewToolWithRawSchema(
	"weather",
	"Get current weather for a location",
	json.RawMessage(`{
		"type": "object",
		"properties": {
			"location": {
				"type": "string",
				"description": "City name",
				"minLength": 2,
				"maxLength": 100
			},
			"units": {
				"type": "string",
				"description": "Temperature units",
				"enum": ["celsius", "fahrenheit"],
				"default": "celsius"
			},
			"forecast_days": {
				"type": "integer",
				"description": "Number of forecast days",
				"minimum": 1,
				"maximum": 7,
				"default": 1
			}
		},
		"required": ["location"],
		"additionalProperties": false
	}`),
)

func HandleWeather(ctx context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	location := request.GetString("location", "")
	units := request.GetString("units", "celsius")
	forecastDays := request.GetInt("forecast_days", 1)

	simulatedTemp := 22.5
	if units == "fahrenheit" {
		simulatedTemp = simulatedTemp*9/5 + 32
	}

	result := map[string]any{
		"location":      location,
		"temperature":   simulatedTemp,
		"units":         units,
		"forecast_days": forecastDays,
		"condition":     "sunny",
		"humidity":      65,
	}

	jsonResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	return mcplib.NewToolResultText(fmt.Sprintf("Weather for %s:\n%s", location, string(jsonResult))), nil
}

var SearchTool = mcplib.NewToolWithRawSchema(
	"search",
	"Search for information",
	json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "Search query",
				"minLength": 1
			},
			"filters": {
				"type": "object",
				"properties": {
					"category": {
						"type": "string",
						"enum": ["docs", "blog", "forum"]
					},
					"date_range": {
						"type": "string",
						"pattern": "^\\d{4}-\\d{2}-\\d{2}$"
					},
					"limit": {
						"type": "integer",
						"minimum": 1,
						"maximum": 50
					}
				}
			}
		},
		"required": ["query"]
	}`),
)

func HandleSearch(ctx context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	query := request.GetString("query", "")

	return mcplib.NewToolResultText(fmt.Sprintf("Search results for: %s", query)), nil
}
