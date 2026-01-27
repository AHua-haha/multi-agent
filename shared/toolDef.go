package shared

import (
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sashabaranov/go-openai"
)

func ConvertToMcpTool(def openai.FunctionDefinition) (mcp.Tool, error) {

	data, err := json.Marshal(def.Parameters)
	if err != nil {
		return mcp.Tool{}, err
	}

	tool := mcp.NewToolWithRawSchema(def.Name, def.Description, data)
	return tool, nil
}
