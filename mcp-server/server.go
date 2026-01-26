package mcp

import (
	"context"
	"fmt"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

var CalculatorTool = mcplib.NewTool("calculator",
	mcplib.WithDescription("Perform basic arithmetic operations (add, subtract, multiply, divide)"),
	mcplib.WithString("operation",
		mcplib.Required(),
		mcplib.Description("The operation to perform: add, subtract, multiply, or divide"),
		mcplib.Enum("add", "subtract", "multiply", "divide"),
	),
	mcplib.WithNumber("a",
		mcplib.Required(),
		mcplib.Description("The first number"),
	),
	mcplib.WithNumber("b",
		mcplib.Required(),
		mcplib.Description("The second number"),
	),
)

func HandleCalculator(ctx context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	op, err := request.RequireString("operation")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	a, err := request.RequireFloat("a")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	b, err := request.RequireFloat("b")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	var result float64

	switch op {
	case "add":
		result = a + b
	case "subtract":
		result = a - b
	case "multiply":
		result = a * b
	case "divide":
		if b == 0 {
			return mcplib.NewToolResultError("cannot divide by zero"), nil
		}
		result = a / b
	}

	return mcplib.NewToolResultText(fmt.Sprintf("Result of %s %.2f and %.2f = %.2f", op, a, b, result)), nil
}
