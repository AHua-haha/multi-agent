package shared

import (
	"bytes"
	"encoding/json"
	"io"
	"os"

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

func ConvertToFunctionDefinition(def mcp.Tool) openai.FunctionDefinition {
	res := openai.FunctionDefinition{
		Name:        def.Name,
		Description: def.Description,
		Parameters:  def.InputSchema,
		Strict:      true,
	}
	return res
}

func CountLines(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// Use a 32KB buffer for efficient reading
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := file.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil
		case err != nil:
			return count, err
		}
	}
}
