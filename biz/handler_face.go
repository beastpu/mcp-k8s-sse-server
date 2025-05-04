package biz

import (
	"encoding/json"
	"fmt"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/ThinkInAIXYZ/go-mcp/server"
)

type ToolHandler interface {
	GetTools() (map[*protocol.Tool]server.ToolHandlerFunc, error)
}

// ParseParams is a generic function used to parse raw request parameters into a specified parameter type
func ParseParams[T any](req *protocol.CallToolRequest) (T, error) {
	var params T
	if err := json.Unmarshal(req.RawArguments, &params); err != nil {
		return params, fmt.Errorf("failed to parse parameters: %v", err)
	}
	return params, nil
}
