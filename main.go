package main

import (
	"flag"
	"log"

	"github.com/beastpu/mcp-k8s-sse-server/biz"
	// Import sub-packages to execute init functions
	_ "github.com/beastpu/mcp-k8s-sse-server/biz/configmap"
	_ "github.com/beastpu/mcp-k8s-sse-server/biz/context"
	_ "github.com/beastpu/mcp-k8s-sse-server/biz/kruise"
	_ "github.com/beastpu/mcp-k8s-sse-server/biz/node"
	_ "github.com/beastpu/mcp-k8s-sse-server/biz/pod"

	"github.com/ThinkInAIXYZ/go-mcp/server"
	"github.com/ThinkInAIXYZ/go-mcp/transport"
)

var (
	mode    string
	address string
)

func main() {
	flag.StringVar(&mode, "mode", "sse", "Transport mode: 'stdio' or 'sse'")
	flag.StringVar(&address, "address", ":8686", "Address for SSE server")
	flag.Parse()

	// Start the server
	if err := Start(); err != nil {
		log.Fatalf("Server startup failed: %v", err)
	}
}

func Start() error {
	var transportServer transport.ServerTransport
	var err error

	switch mode {
	case "stdio":
		// Use standard input/output for transport
		transportServer = transport.NewStdioServerTransport()
		log.Println("Starting in stdio mode")
	case "sse":
		// Use SSE for transport
		transportServer, err = transport.NewSSEServerTransport(address)
		if err != nil {
			return err
		}
		log.Printf("Starting in SSE mode on %s\n", address)
	default:
		log.Fatalf("Invalid mode: %s. Must be 'stdio' or 'sse'\n", mode)
	}

	// Initialize MCP server
	mcpServer, err := server.NewServer(transportServer)
	if err != nil {
		return err
	}

	// Register tools
	for _, fn := range biz.ToolRegisterFactory {
		if err := fn(mcpServer); err != nil {
			return err
		}
	}

	// Start server
	if err = mcpServer.Run(); err != nil {
		return err
	}
	return nil
}
