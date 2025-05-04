package node

import (
	"context"
	"fmt"
	"lilithgames/mcp-k8s-server/biz"

	kubeclient "lilithgames/mcp-k8s-server/biz/clientset"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/ThinkInAIXYZ/go-mcp/server"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func init() {
	handler, err := NewNodeHandler()
	if err != nil {
		panic(err)
	}
	biz.RegisterHandler(handler)
}

func NewNodeHandler() (*NodeHandler, error) {
	tools := make(map[*protocol.Tool]server.ToolHandlerFunc)
	n := &NodeHandler{
		tools: tools,
	}

	// Node cordon tool
	cordonNodeTool, err := protocol.NewTool(
		"cordon_node",
		"Mark Kubernetes Node as Unschedulable",
		struct {
			NodeName string `json:"nodeName" description:"Name of the node" required:"true"`
		}{},
	)
	if err != nil {
		return nil, err
	}

	// Node uncordon tool
	uncordonNodeTool, err := protocol.NewTool(
		"uncordon_node",
		"Mark Kubernetes Node as Schedulable",
		struct {
			NodeName string `json:"nodeName" description:"Name of the node" required:"true"`
		}{},
	)
	if err != nil {
		return nil, err
	}

	// Describe node tool
	describeNodeTool, err := protocol.NewTool(
		"describe_node",
		"Get Detailed Information of a Kubernetes Node",
		struct {
			NodeName string `json:"nodeName" description:"Name of the node" required:"true"`
		}{},
	)
	if err != nil {
		return nil, err
	}

	// List nodes tool
	listNodesTool, err := protocol.NewTool(
		"list_nodes",
		"List All Kubernetes Nodes",
		struct {
			LabelSelector string `json:"labelSelector" description:"Label selector for filtering nodes"`
		}{},
	)
	if err != nil {
		return nil, err
	}

	tools[cordonNodeTool] = n.cordonNode
	tools[uncordonNodeTool] = n.uncordonNode
	tools[describeNodeTool] = n.describe
	tools[listNodesTool] = n.list

	return n, nil
}

type NodeHandler struct {
	tools map[*protocol.Tool]server.ToolHandlerFunc
}

func (n *NodeHandler) GetTools() (map[*protocol.Tool]server.ToolHandlerFunc, error) {
	return n.tools, nil
}

// Handle cordon_node tool
func (n *NodeHandler) cordonNode(_ context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	params, err := biz.ParseParams[NodeParams](req)
	if err != nil {
		return nil, err
	}

	// Get the latest clientset
	clientset, err := kubeclient.GetKubeClient()
	if err != nil {
		return nil, err
	}

	// Mark node as unschedulable
	err = n.markNodeAsUnschedulableState(clientset, params.NodeName, true)
	if err != nil {
		return nil, err
	}

	return &protocol.CallToolResult{
		Content: []protocol.Content{
			protocol.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Node %s has been marked as unschedulable", params.NodeName),
			},
		},
	}, nil
}

// Handle uncordon_node tool
func (n *NodeHandler) uncordonNode(_ context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	params, err := biz.ParseParams[NodeParams](req)
	if err != nil {
		return nil, err
	}

	// Get the latest clientset
	clientset, err := kubeclient.GetKubeClient()
	if err != nil {
		return nil, err
	}

	// Mark node as schedulable
	err = n.markNodeAsUnschedulableState(clientset, params.NodeName, false)
	if err != nil {
		return nil, err
	}

	return &protocol.CallToolResult{
		Content: []protocol.Content{
			protocol.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Node %s has been marked as schedulable", params.NodeName),
			},
		},
	}, nil
}

func (n *NodeHandler) describe(_ context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	params, err := biz.ParseParams[NodeParams](req)
	if err != nil {
		return nil, err
	}

	// Get the latest clientset
	clientset, err := kubeclient.GetKubeClient()
	if err != nil {
		return nil, err
	}

	nodeInfo, err := n.describeNode(clientset, params.NodeName)
	if err != nil {
		return nil, err
	}

	return &protocol.CallToolResult{
		Content: []protocol.Content{
			protocol.TextContent{
				Type: "text",
				Text: *nodeInfo,
			},
		},
	}, nil
}

// Handle describe_node tool
func (n *NodeHandler) describeNode(clientset kubernetes.Interface, nodeName string) (*string, error) {
	// Get node information
	node, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// Format node information using biz.FormatNodeInfoTable
	nodeInfo := biz.FormatNodeInfoTable(node)
	return &nodeInfo, nil
}

// Handle list_nodes tool
func (n *NodeHandler) list(_ context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	params, err := biz.ParseParams[NodeListParams](req)
	if err != nil {
		return nil, err
	}

	// Get the latest clientset
	clientset, err := kubeclient.GetKubeClient()
	if err != nil {
		return nil, err
	}

	result, err := n.listNodes(clientset, params.LabelSelector)
	if err != nil {
		return nil, err
	}

	return &protocol.CallToolResult{
		Content: []protocol.Content{
			protocol.TextContent{
				Type: "text",
				Text: *result,
			},
		},
	}, nil
}

func (n *NodeHandler) listNodes(clientset kubernetes.Interface, labelSelector string) (*string, error) {
	// Get node list
	listOptions := metav1.ListOptions{}
	if labelSelector != "" {
		listOptions.LabelSelector = labelSelector
	}

	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), listOptions)
	if err != nil {
		return nil, err
	}

	// Format node list using biz.FormatNodesTable to match kubectl get nodes output
	result := biz.FormatNodesTable(nodes.Items)

	return &result, nil
}

// Mark node as unschedulable
func (n *NodeHandler) markNodeAsUnschedulableState(clientset kubernetes.Interface, nodeName string, unscheduleable bool) error {
	// Get the node
	node, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Check if the node is already unschedulable
	if node.Spec.Unschedulable == unscheduleable {
		return fmt.Errorf("node %s is already marked unscheduable:%v", nodeName, unscheduleable)
	}

	// Make a copy of the node to avoid modifying the original
	newNode := node.DeepCopy()
	newNode.Spec.Unschedulable = unscheduleable

	// Update the node
	_, err = clientset.CoreV1().Nodes().Update(context.TODO(), newNode, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

