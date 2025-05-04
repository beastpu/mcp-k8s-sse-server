package biz

import (
	"github.com/ThinkInAIXYZ/go-mcp/server"
)

// Tool register factory list
var ToolRegisterFactory = make([]func(mcpServer *server.Server) error, 0)

// ToolRegister register tool handler function
func ToolRegister(fn func(mcpServer *server.Server) error) {
	ToolRegisterFactory = append(ToolRegisterFactory, fn)
}

// RegisterHandler register tool handler
func RegisterHandler(handler ToolHandler) {
	ToolRegisterFactory = append(ToolRegisterFactory, registerHandler(handler))
}

// registerHandler convert handler to register function
func registerHandler(handler ToolHandler) func(mcpServer *server.Server) error {
	return func(mcpServer *server.Server) error {
		tools, err := handler.GetTools()
		if err != nil {
			return err
		}
		for tool, handler := range tools {
			mcpServer.RegisterTool(tool, handler)
		}
		return nil
	}
}

