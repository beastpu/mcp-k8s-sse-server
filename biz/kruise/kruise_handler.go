package kruise

import (
	"context"
	"fmt"
	"strconv"

	"github.com/beastpu/mcp-k8s-sse-server/biz"

	kubeclient "github.com/beastpu/mcp-k8s-sse-server/biz/clientset"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/ThinkInAIXYZ/go-mcp/server"
	appsv1alpha1 "github.com/openkruise/kruise-api/apps/v1alpha1"
	appsv1beta1 "github.com/openkruise/kruise-api/apps/v1beta1"
	kruiseclientset "github.com/openkruise/kruise-api/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	handler, err := NewKruiseHandler()
	if err != nil {
		panic(err)
	}
	biz.RegisterHandler(handler)
}

func NewKruiseHandler() (*KruiseHandler, error) {
	tools := make(map[*protocol.Tool]server.ToolHandlerFunc)
	k := &KruiseHandler{
		tools: tools,
	}

	// List AdvancedStatefulSets tool
	listAdvancedStatefulSetsTool, err := protocol.NewTool(
		"list_advanced_statefulsets",
		"List AdvancedStatefulSets",
		struct {
			Namespace     string `json:"namespace" description:"Namespace of the resource, default is 'default'"`
			AllNamespaces bool   `json:"allNamespaces" description:"Whether to list resources in all namespaces"`
		}{},
	)
	if err != nil {
		return nil, err
	}

	// List CloneSets tool
	listCloneSetsTool, err := protocol.NewTool(
		"list_clonesets",
		"List CloneSets",
		struct {
			Namespace     string `json:"namespace" description:"Namespace of the resource, default is 'default'"`
			AllNamespaces bool   `json:"allNamespaces" description:"Whether to list resources in all namespaces"`
		}{},
	)
	if err != nil {
		return nil, err
	}

	// Scale resource tool
	scaleResourceTool, err := protocol.NewTool(
		"scale_kruise_resource",
		"Scale OpenKruise Resource Replicas",
		struct {
			ResourceType string `json:"resourceType" description:"Resource type, e.g. 'advancedstatefulset' or 'cloneset'" required:"true"`
			Namespace    string `json:"namespace" description:"Namespace of the resource, default is 'default'"`
			ResourceName string `json:"resourceName" description:"Name of the resource to scale" required:"true"`
			Replicas     string `json:"replicas" description:"Number of replicas to scale to" required:"true"`
		}{},
	)
	if err != nil {
		return nil, err
	}

	// Generic scale tool
	scaleTool, err := protocol.NewTool(
		"scale",
		"Scale OpenKruise Resource Replicas",
		struct {
			ResourceType string `json:"resourceType" description:"Resource type, e.g. 'advancedstatefulset' or 'cloneset'" required:"true"`
			Namespace    string `json:"namespace" description:"Namespace of the resource, default is 'default'"`
			ResourceName string `json:"resourceName" description:"Name of the resource to scale" required:"true"`
			Replicas     string `json:"replicas" description:"Number of replicas to scale to" required:"true"`
		}{},
	)
	if err != nil {
		return nil, err
	}

	// Describe AdvancedStatefulSet tool
	describeAdvancedStatefulSetTool, err := protocol.NewTool(
		"describe_advanced_statefulset",
		"Describe AdvancedStatefulSet",
		struct {
			Namespace string `json:"namespace" description:"Namespace of the resource, default is 'default'"`
			Name      string `json:"name" description:"Name of the resource" required:"true"`
		}{},
	)
	if err != nil {
		return nil, err
	}

	// Describe CloneSet tool
	describeCloneSetTool, err := protocol.NewTool(
		"describe_cloneset",
		"Describe CloneSet",
		struct {
			Namespace string `json:"namespace" description:"Namespace of the resource, default is 'default'"`
			Name      string `json:"name" description:"Name of the resource" required:"true"`
		}{},
	)
	if err != nil {
		return nil, err
	}

	// Register tool handlers
	tools[listAdvancedStatefulSetsTool] = k.listAdvancedStatefulSets
	tools[listCloneSetsTool] = k.listCloneSets
	tools[scaleResourceTool] = k.scaleResource
	tools[scaleTool] = k.scale
	tools[describeAdvancedStatefulSetTool] = k.describeAdvancedStatefulSet
	tools[describeCloneSetTool] = k.describeCloneSet

	return k, nil
}

type KruiseHandler struct {
	tools map[*protocol.Tool]server.ToolHandlerFunc
}

func (k *KruiseHandler) GetTools() (map[*protocol.Tool]server.ToolHandlerFunc, error) {
	return k.tools, nil
}

// Get the latest kruiseClient
func (k *KruiseHandler) getKruiseClient() (kruiseclientset.Interface, error) {
	return kubeclient.GetKruiseClient()
}

// Handle list_advanced_statefulsets tool
func (k *KruiseHandler) listAdvancedStatefulSets(_ context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	params, err := biz.ParseParams[KruiseNamespaceParams](req)
	if err != nil {
		return nil, err
	}

	// Set default namespace
	if params.Namespace == "" {
		params.Namespace = "default"
	}

	// List AdvancedStatefulSets
	output, err := k.listAdvancedStatefulSetsInternal(params.Namespace, params.AllNamespaces)
	if err != nil {
		return nil, err
	}

	return &protocol.CallToolResult{
		Content: []protocol.Content{
			protocol.TextContent{
				Type: "text",
				Text: output,
			},
		},
	}, nil
}

// Handle list_clonesets tool
func (k *KruiseHandler) listCloneSets(_ context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	params, err := biz.ParseParams[KruiseNamespaceParams](req)
	if err != nil {
		return nil, err
	}

	// Set default namespace
	if params.Namespace == "" {
		params.Namespace = "default"
	}

	// List CloneSets
	output, err := k.listCloneSetsInternal(params.Namespace, params.AllNamespaces)
	if err != nil {
		return nil, err
	}

	return &protocol.CallToolResult{
		Content: []protocol.Content{
			protocol.TextContent{
				Type: "text",
				Text: output,
			},
		},
	}, nil
}

// Handle scale_kruise_resource tool
func (k *KruiseHandler) scaleResource(_ context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	params, err := biz.ParseParams[KruiseScaleParams](req)
	if err != nil {
		return nil, err
	}

	// Set default namespace
	if params.Namespace == "" {
		params.Namespace = "default"
	}

	// Convert replicas to int32
	replicasInt, err := strconv.Atoi(params.Replicas)
	if err != nil {
		return nil, fmt.Errorf("could not convert replicas parameter '%s' to integer: %v", params.Replicas, err)
	}
	replicas := int32(replicasInt)

	// Choose appropriate handling function based on resource type
	var output string

	switch params.ResourceType {
	case "advancedstatefulset", "advancedstatefulsets", "asts":
		output, err = k.scaleAdvancedStatefulSet(params.Namespace, params.ResourceName, replicas)
		if err != nil {
			return nil, err
		}

	case "cloneset", "clonesets":
		output, err = k.scaleCloneSet(params.Namespace, params.ResourceName, replicas)
		if err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("unsupported resource type: %s", params.ResourceType)
	}

	return &protocol.CallToolResult{
		Content: []protocol.Content{
			protocol.TextContent{
				Type: "text",
				Text: output,
			},
		},
	}, nil
}

// Handle generic scale tool
func (k *KruiseHandler) scale(_ context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	params, err := biz.ParseParams[KruiseScaleParams](req)
	if err != nil {
		return nil, err
	}

	// Set default namespace
	if params.Namespace == "" {
		params.Namespace = "default"
	}

	// Convert replicas to int32
	replicasInt, err := strconv.Atoi(params.Replicas)
	if err != nil {
		return nil, fmt.Errorf("could not convert replicas parameter '%s' to integer: %v", params.Replicas, err)
	}
	replicas := int32(replicasInt)

	// Choose appropriate handling function based on resource type
	var output string

	switch params.ResourceType {
	case "advancedstatefulset", "advancedstatefulsets", "asts":
		output, err = k.scaleAdvancedStatefulSet(params.Namespace, params.ResourceName, replicas)
		if err != nil {
			return nil, err
		}

	case "cloneset", "clonesets":
		output, err = k.scaleCloneSet(params.Namespace, params.ResourceName, replicas)
		if err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("unsupported resource type: %s", params.ResourceType)
	}

	return &protocol.CallToolResult{
		Content: []protocol.Content{
			protocol.TextContent{
				Type: "text",
				Text: output,
			},
		},
	}, nil
}

// Handle describe_advanced_statefulset tool
func (k *KruiseHandler) describeAdvancedStatefulSet(_ context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	params, err := biz.ParseParams[KruiseDescribeParams](req)
	if err != nil {
		return nil, err
	}

	// Set default namespace
	if params.Namespace == "" {
		params.Namespace = "default"
	}

	// Describe AdvancedStatefulSet
	output, err := k.describeAdvancedStatefulSetInternal(params.Namespace, params.Name)
	if err != nil {
		return nil, err
	}

	return &protocol.CallToolResult{
		Content: []protocol.Content{
			protocol.TextContent{
				Type: "text",
				Text: output,
			},
		},
	}, nil
}

// Handle describe_cloneset tool
func (k *KruiseHandler) describeCloneSet(_ context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	params, err := biz.ParseParams[KruiseDescribeParams](req)
	if err != nil {
		return nil, err
	}

	// Set default namespace
	if params.Namespace == "" {
		params.Namespace = "default"
	}

	// Describe CloneSet
	output, err := k.describeCloneSetInternal(params.Namespace, params.Name)
	if err != nil {
		return nil, err
	}

	return &protocol.CallToolResult{
		Content: []protocol.Content{
			protocol.TextContent{
				Type: "text",
				Text: output,
			},
		},
	}, nil
}

// Scale AdvancedStatefulSet replicas
func (k *KruiseHandler) scaleAdvancedStatefulSet(namespace, name string, replicas int32) (string, error) {
	kruiseClient, err := k.getKruiseClient()
	if err != nil {
		return "", err
	}

	// Get the AdvancedStatefulSet
	ast, err := kruiseClient.AppsV1beta1().StatefulSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	// Update the replicas
	ast.Spec.Replicas = &replicas
	_, err = kruiseClient.AppsV1beta1().StatefulSets(namespace).Update(context.TODO(), ast, metav1.UpdateOptions{})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully scaled AdvancedStatefulSet %s in namespace %s to %d replicas", name, namespace, replicas), nil
}

// Scale CloneSet replicas
func (k *KruiseHandler) scaleCloneSet(namespace, name string, replicas int32) (string, error) {
	kruiseClient, err := k.getKruiseClient()
	if err != nil {
		return "", err
	}

	// Get the CloneSet
	cloneSet, err := kruiseClient.AppsV1alpha1().CloneSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	// Update the replicas
	cloneSet.Spec.Replicas = &replicas
	_, err = kruiseClient.AppsV1alpha1().CloneSets(namespace).Update(context.TODO(), cloneSet, metav1.UpdateOptions{})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully scaled CloneSet %s in namespace %s to %d replicas", name, namespace, replicas), nil
}

// Get detailed information about an AdvancedStatefulSet
func (k *KruiseHandler) describeAdvancedStatefulSetInternal(namespace, name string) (string, error) {
	kruiseClient, err := k.getKruiseClient()
	if err != nil {
		return "", err
	}

	// Get the AdvancedStatefulSet
	ast, err := kruiseClient.AppsV1beta1().StatefulSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	return biz.FormatAdvancedStatefulSetDetail(ast), nil
}

// Get detailed information about a CloneSet
func (k *KruiseHandler) describeCloneSetInternal(namespace, name string) (string, error) {
	kruiseClient, err := k.getKruiseClient()
	if err != nil {
		return "", err
	}

	// Get the CloneSet
	cloneSet, err := kruiseClient.AppsV1alpha1().CloneSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	return biz.FormatCloneSetDetail(cloneSet), nil
}

// List AdvancedStatefulSets in a namespace or across all namespaces
func (k *KruiseHandler) listAdvancedStatefulSetsInternal(namespace string, allNamespaces bool) (string, error) {
	kruiseClient, err := k.getKruiseClient()
	if err != nil {
		return "", err
	}

	var astsList *appsv1beta1.StatefulSetList
	var listErr error

	if allNamespaces {
		astsList, listErr = kruiseClient.AppsV1beta1().StatefulSets("").List(context.TODO(), metav1.ListOptions{})
	} else {
		astsList, listErr = kruiseClient.AppsV1beta1().StatefulSets(namespace).List(context.TODO(), metav1.ListOptions{})
	}

	if listErr != nil {
		return "", listErr
	}

	if len(astsList.Items) == 0 {
		if allNamespaces {
			return "No AdvancedStatefulSets found in any namespace", nil
		} else {
			return fmt.Sprintf("No AdvancedStatefulSets found in namespace %s", namespace), nil
		}
	}

	// Format the output using biz.FormatAdvancedStatefulSetsTable
	return biz.FormatAdvancedStatefulSetsTable(astsList.Items), nil
}

// List CloneSets in a namespace or across all namespaces
func (k *KruiseHandler) listCloneSetsInternal(namespace string, allNamespaces bool) (string, error) {
	kruiseClient, err := k.getKruiseClient()
	if err != nil {
		return "", err
	}

	var cloneSetsList *appsv1alpha1.CloneSetList
	var listErr error

	if allNamespaces {
		cloneSetsList, listErr = kruiseClient.AppsV1alpha1().CloneSets("").List(context.TODO(), metav1.ListOptions{})
	} else {
		cloneSetsList, listErr = kruiseClient.AppsV1alpha1().CloneSets(namespace).List(context.TODO(), metav1.ListOptions{})
	}

	if listErr != nil {
		return "", listErr
	}

	if len(cloneSetsList.Items) == 0 {
		if allNamespaces {
			return "No CloneSets found in any namespace", nil
		} else {
			return fmt.Sprintf("No CloneSets found in namespace %s", namespace), nil
		}
	}

	// Format the output using biz.FormatCloneSetsTable
	return biz.FormatCloneSetsTable(cloneSetsList.Items), nil
}
