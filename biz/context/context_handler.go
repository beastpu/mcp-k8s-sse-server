package context

import (
	"context"
	"fmt"
	"lilithgames/mcp-k8s-server/biz"
	"strings"

	kubeclient "lilithgames/mcp-k8s-server/biz/clientset"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/ThinkInAIXYZ/go-mcp/server"
	"k8s.io/client-go/tools/clientcmd"
)

func init() {
	handler, err := NewContextHandler()
	if err != nil {
		panic(err)
	}
	biz.RegisterHandler(handler)
}

func NewContextHandler() (*ContextHandler, error) {
	tools := make(map[*protocol.Tool]server.ToolHandlerFunc)
	c := &ContextHandler{
		tools: tools,
	}

	// Set kubeconfig path tool
	setKubeconfigPathTool, err := protocol.NewTool(
		"set_kubeconfig_path",
		"Set Custom Kubeconfig File Path",
		struct {
			KubeconfigPath string `json:"kubeconfigPath" description:"Path to the kubeconfig file" required:"true"`
		}{},
	)
	if err != nil {
		return nil, err
	}

	// Get current context tool
	getCurrentContextTool, err := protocol.NewTool(
		"get_current_context",
		"Get Current Kubernetes Context",
		struct{}{},
	)
	if err != nil {
		return nil, err
	}

	// List contexts tool
	listContextsTool, err := protocol.NewTool(
		"list_contexts",
		"List All Available Kubernetes Contexts",
		struct{}{},
	)
	if err != nil {
		return nil, err
	}

	// Switch context tool
	switchContextTool, err := protocol.NewTool(
		"switch_context",
		"Switch to Specified Kubernetes Context",
		struct {
			ContextName string `json:"contextName" description:"Name of the context to switch to" required:"true"`
		}{},
	)
	if err != nil {
		return nil, err
	}

	tools[setKubeconfigPathTool] = c.setKubeconfigPath
	tools[getCurrentContextTool] = c.getCurrentContext
	tools[listContextsTool] = c.listContexts
	tools[switchContextTool] = c.switchContext

	return c, nil
}

type ContextHandler struct {
	tools map[*protocol.Tool]server.ToolHandlerFunc
}

func (c *ContextHandler) GetTools() (map[*protocol.Tool]server.ToolHandlerFunc, error) {
	return c.tools, nil
}

// Handle set_kubeconfig_path tool
func (c *ContextHandler) setKubeconfigPath(_ context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	params, err := biz.ParseParams[KubeconfigPathParams](req)
	if err != nil {
		return nil, err
	}

	result, err := c.setKubeconfigPathInternal(params.KubeconfigPath)
	if err != nil {
		return nil, err
	}

	return &protocol.CallToolResult{
		Content: []protocol.Content{
			protocol.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}, nil
}

// Handle get_current_context tool
func (c *ContextHandler) getCurrentContext(_ context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	currentCtx, err := kubeclient.GetCurrentContext()
	if err != nil {
		return nil, err
	}

	return &protocol.CallToolResult{
		Content: []protocol.Content{
			protocol.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Current context: %s", currentCtx),
			},
		},
	}, nil
}

// Handle list_contexts tool
func (c *ContextHandler) listContexts(_ context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	contexts, err := c.listContextsInternal()
	if err != nil {
		return nil, err
	}

	return &protocol.CallToolResult{
		Content: []protocol.Content{
			protocol.TextContent{
				Type: "text",
				Text: contexts,
			},
		},
	}, nil
}

// Handle switch_context tool
func (c *ContextHandler) switchContext(_ context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	params, err := biz.ParseParams[ContextNameParams](req)
	if err != nil {
		return nil, err
	}

	result, err := c.switchContextInternal(params.ContextName)
	if err != nil {
		return nil, err
	}

	return &protocol.CallToolResult{
		Content: []protocol.Content{
			protocol.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}, nil
}

// Set custom kubeconfig path
func (c *ContextHandler) setKubeconfigPathInternal(kubeconfigPath string) (string, error) {
	fmt.Printf("Debug - Attempting to set kubeconfig path: %s\n", kubeconfigPath)

	// Validate and try to fix kubeconfig file
	if err := kubeclient.ValidateAndFixKubeconfig(kubeconfigPath); err != nil {
		return "", fmt.Errorf("kubeconfig validation failed: %v", err)
	}

	// Save new kubeconfig path
	kubeclient.SetCustomKubeconfigPath(kubeconfigPath)

	// Reset current context, as the new kubeconfig may have a different current context
	kubeclient.ResetCurrentContext()

	// Clear client cache
	kubeclient.ClearClientCache()

	// Try to get the current context from the new kubeconfig file
	ctx, err := kubeclient.GetCurrentContext()
	if err != nil {
		return "", fmt.Errorf("could not get current context after setting kubeconfig path: %v", err)
	}

	// Output debug info
	fmt.Printf("Debug - Context after setting kubeconfig path: %s\n", ctx)

	// Get new kubeconfig configuration information
	configAfterSet, err := kubeclient.GetKubeConfig()
	if err != nil {
		return "", fmt.Errorf("could not reload kubeconfig file: %v", err)
	}

	// Handle case where ctx is empty
	contextInfo := ctx
	if ctx == "" {
		if len(configAfterSet.Contexts) > 0 {
			// If there are available contexts but current context is empty,
			// try to automatically switch to the first context
			for name := range configAfterSet.Contexts {
				// Update the current context in config
				configAfterSet.CurrentContext = name

				// Create configuration accessor
				configAccess := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeclient.GetCustomKubeconfigPath()}

				// Write the modified configuration back to file
				if writeErr := clientcmd.ModifyConfig(configAccess, *configAfterSet, true); writeErr != nil {
					return "", fmt.Errorf("failed to save kubeconfig file: %v", writeErr)
				}

				// Set current context
				kubeclient.SetCurrentContext(name)

				// Clear client cache
				kubeclient.ClearClientCache()

				contextInfo = name
				fmt.Printf("Debug - Automatically switched to first available context: %s\n", name)
				break
			}
		}
	}

	return fmt.Sprintf("Kubeconfig path has been set to %s, current context: %s", kubeconfigPath, contextInfo), nil
}

// List all available contexts
func (c *ContextHandler) listContextsInternal() (string, error) {
	// Get kubeconfig configuration
	config, err := kubeclient.GetKubeConfig()
	if err != nil {
		return "", err
	}

	// Get current context
	currentCtx, err := kubeclient.GetCurrentContext()
	if err != nil {
		return "", err
	}

	// Format output
	var sb strings.Builder
	sb.WriteString("Available Kubernetes contexts:\n")
	sb.WriteString("----------------------------\n")

	if len(config.Contexts) == 0 {
		sb.WriteString("No contexts found in kubeconfig\n")
	} else {
		for name, ctx := range config.Contexts {
			// Mark current context with an asterisk
			if name == currentCtx {
				sb.WriteString(fmt.Sprintf("* %s\n", name))
			} else {
				sb.WriteString(fmt.Sprintf("  %s\n", name))
			}

			// Add cluster and user information
			sb.WriteString(fmt.Sprintf("    Cluster: %s\n", ctx.Cluster))
			sb.WriteString(fmt.Sprintf("    User: %s\n", ctx.AuthInfo))
			sb.WriteString(fmt.Sprintf("    Namespace: %s\n", ctx.Namespace))
			sb.WriteString("\n")
		}
	}

	// Add path info
	customPath := kubeclient.GetCustomKubeconfigPath()
	if customPath != "" {
		sb.WriteString(fmt.Sprintf("\nUsing custom kubeconfig: %s\n", customPath))
	} else {
		sb.WriteString("\nUsing default kubeconfig path\n")
	}

	return sb.String(), nil
}

// Switch to specified context
func (c *ContextHandler) switchContextInternal(contextName string) (string, error) {
	// Get kubeconfig configuration
	config, err := kubeclient.GetKubeConfig()
	if err != nil {
		return "", err
	}

	// Check if the context exists
	if _, exists := config.Contexts[contextName]; !exists {
		return "", fmt.Errorf("context '%s' does not exist in kubeconfig", contextName)
	}

	// Get current context for comparison
	currentCtx, err := kubeclient.GetCurrentContext()
	if err != nil {
		return "", err
	}

	// If already using the requested context, just return
	if currentCtx == contextName {
		return fmt.Sprintf("Already using context '%s'", contextName), nil
	}

	// Set the current context in configuration
	config.CurrentContext = contextName

	// Create configuration accessor
	var configAccess clientcmd.ConfigAccess
	customPath := kubeclient.GetCustomKubeconfigPath()
	if customPath != "" {
		configAccess = &clientcmd.ClientConfigLoadingRules{ExplicitPath: customPath}
	} else {
		configAccess = clientcmd.NewDefaultClientConfigLoadingRules()
	}

	// Update kubeconfig file
	if err := clientcmd.ModifyConfig(configAccess, *config, true); err != nil {
		return "", fmt.Errorf("failed to modify kubeconfig: %v", err)
	}

	// Set current context in memory
	kubeclient.SetCurrentContext(contextName)

	// Clear client cache to ensure using the new context
	kubeclient.ClearClientCache()

	return fmt.Sprintf("Switched to context '%s'", contextName), nil
}
