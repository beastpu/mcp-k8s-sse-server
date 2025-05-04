package configmap

import (
	"context"
	"fmt"

	"github.com/beastpu/mcp-k8s-sse-server/biz"
	kubeclient "github.com/beastpu/mcp-k8s-sse-server/biz/clientset"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/ThinkInAIXYZ/go-mcp/server"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func init() {
	handler, err := NewConfigMapHandler()
	if err != nil {
		panic(err)
	}
	biz.RegisterHandler(handler)
}

func NewConfigMapHandler() (*ConfigMapHandler, error) {
	tools := make(map[*protocol.Tool]server.ToolHandlerFunc)
	c := &ConfigMapHandler{
		tools: tools,
	}

	// Get ConfigMap content tool
	getConfigMapTool, err := protocol.NewTool(
		"get_configmap",
		"Get ConfigMap Content",
		struct {
			Namespace     string `json:"namespace" description:"Namespace of the ConfigMap, default is 'default'" required:"true"`
			ConfigMapName string `json:"configMapName" description:"Name of the ConfigMap" required:"true"`
		}{},
	)
	if err != nil {
		return nil, err
	}

	// List ConfigMaps tool
	listConfigMapsTool, err := protocol.NewTool(
		"list_configmaps",
		"List All ConfigMaps",
		struct {
			Namespace string `json:"namespace" description:"Namespace of ConfigMaps, if empty will list from all namespaces" required:"false"`
		}{},
	)
	if err != nil {
		return nil, err
	}

	tools[getConfigMapTool] = c.getConfigMap
	tools[listConfigMapsTool] = c.listConfigMaps
	return c, nil
}

type ConfigMapHandler struct {
	tools map[*protocol.Tool]server.ToolHandlerFunc
}

func (c *ConfigMapHandler) GetTools() (map[*protocol.Tool]server.ToolHandlerFunc, error) {
	return c.tools, nil
}

// Handle get_configmap tool
func (c *ConfigMapHandler) getConfigMap(_ context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	params, err := biz.ParseParams[ConfigMapParams](req)
	if err != nil {
		return nil, err
	}

	// Set default namespace
	if params.Namespace == "" {
		params.Namespace = "default"
	}

	// Get the latest clientset
	clientset, err := kubeclient.GetKubeClient()
	if err != nil {
		return nil, err
	}

	// Get ConfigMap
	configMapContent, err := c.getConfigMapContent(clientset, params.Namespace, params.ConfigMapName)
	if err != nil {
		return nil, err
	}

	return &protocol.CallToolResult{
		Content: []protocol.Content{
			protocol.TextContent{
				Type: "text",
				Text: configMapContent,
			},
		},
	}, nil
}

// Handle list_configmaps tool
func (c *ConfigMapHandler) listConfigMaps(_ context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	params, err := biz.ParseParams[ListConfigMapsParams](req)
	if err != nil {
		return nil, err
	}

	// Get the latest clientset
	clientset, err := kubeclient.GetKubeClient()
	if err != nil {
		return nil, err
	}

	// List ConfigMaps
	configMapsList, err := c.listConfigMapsContent(clientset, params.Namespace)
	if err != nil {
		return nil, err
	}

	return &protocol.CallToolResult{
		Content: []protocol.Content{
			protocol.TextContent{
				Type: "text",
				Text: configMapsList,
			},
		},
	}, nil
}

// Get ConfigMap content
func (c *ConfigMapHandler) getConfigMapContent(clientset kubernetes.Interface, namespace, name string) (string, error) {

	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get ConfigMap %s in namespace %s: %v", name, namespace, err)
	}

	return biz.FormatConfigMapDetail(configMap), nil
}

// List ConfigMaps content
func (c *ConfigMapHandler) listConfigMapsContent(clientset kubernetes.Interface, namespace string) (string, error) {
	if namespace != "" {
		// List ConfigMaps in the specified namespace
		configMaps, err := clientset.CoreV1().ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to list ConfigMaps in namespace %s: %v", namespace, err)
		}

		// Use formatting tool from biz package
		return fmt.Sprintf("ConfigMaps in namespace %s:\n\n%s",
			namespace,
			biz.FormatConfigMapsTable(configMaps.Items)), nil
	} else {
		// List ConfigMaps across all namespaces
		configMaps, err := clientset.CoreV1().ConfigMaps("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to list ConfigMaps across all namespaces: %v", err)
		}

		return fmt.Sprintf("ConfigMaps across all namespaces:\n\n%s",
			biz.FormatConfigMapsTable(configMaps.Items)), nil
	}
}
