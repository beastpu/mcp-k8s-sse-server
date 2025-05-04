package pod

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/beastpu/mcp-k8s-sse-server/biz"

	kubeclient "github.com/beastpu/mcp-k8s-sse-server/biz/clientset"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/ThinkInAIXYZ/go-mcp/server"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

func init() {
	handler, err := NewPodHandler()
	if err != nil {
		panic(err)
	}
	biz.RegisterHandler(handler)
}

func NewPodHandler() (*PodHandler, error) {
	tools := make(map[*protocol.Tool]server.ToolHandlerFunc)
	p := &PodHandler{
		tools: tools,
	}

	getPodLogsTool, err := protocol.NewTool(
		"get_pod_logs",
		"Get Pod Logs",
		struct {
			Namespace string `json:"namespace" description:"Namespace of the Pod" required:"true"`
			PodName   string `json:"podName" description:"Name of the Pod" required:"true"`
			Container string `json:"container" description:"Name of the container to get logs from"`
		}{},
	)
	if err != nil {
		return nil, err
	}

	deletePodTool, err := protocol.NewTool(
		"delete_pod",
		"Delete Pod",
		struct {
			Namespace string `json:"namespace" description:"Namespace of the resource, default is 'default'"`
			PodName   string `json:"podName" description:"Name of the Pod to delete" required:"true"`
			Force     bool   `json:"force" description:"Force delete (only applicable to Pod)"`
		}{},
	)
	if err != nil {
		return nil, err
	}

	// Pod command execution tool
	execCommandTool, err := protocol.NewTool(
		"exec_command_in_pod",
		"Execute Command in Pod",
		struct {
			Context   string `json:"context" description:"Kubernetes cluster context name" required:"true"`
			Namespace string `json:"namespace" description:"Namespace of the Pod" required:"true"`
			PodName   string `json:"podName" description:"Name of the Pod" required:"true"`
			Command   string `json:"command" description:"Command to execute" required:"true"`
		}{},
	)
	if err != nil {
		return nil, err
	}

	// Describe Pod tool
	describePodTool, err := protocol.NewTool(
		"describe_pod",
		"Get Detailed Pod Information",
		struct {
			Namespace string `json:"namespace" description:"Namespace of the Pod, default is 'default'"`
			PodName   string `json:"podName" description:"Name of the Pod" required:"true"`
		}{},
	)
	if err != nil {
		return nil, err
	}

	// List Pods tool
	listPodsTool, err := protocol.NewTool(
		"list_pods",
		"List Pods in a Namespace",
		struct {
			Namespace     string `json:"namespace" description:"Namespace of Pods, default is 'default'"`
			LabelSelector string `json:"labelSelector" description:"Label selector for filtering Pods"`
			AllNamespaces bool   `json:"allNamespaces" description:"List Pods in all namespaces"`
		}{},
	)
	if err != nil {
		return nil, err
	}

	tools[getPodLogsTool] = p.getLogs
	tools[deletePodTool] = p.delete
	tools[execCommandTool] = p.execCommand
	tools[describePodTool] = p.describePod
	tools[listPodsTool] = p.listPods
	return p, nil
}

type PodHandler struct {
	tools map[*protocol.Tool]server.ToolHandlerFunc
}

func (p *PodHandler) GetTools() (map[*protocol.Tool]server.ToolHandlerFunc, error) {
	return p.tools, nil
}

// Handle get_pod_logs tool
func (p *PodHandler) getLogs(_ context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	params, err := biz.ParseParams[podParams](req)
	if err != nil {
		return nil, err
	}
	clientset, err := kubeclient.GetKubeClient()
	if err != nil {
		return nil, err
	}
	// Get Pod logs
	logs, err := p.getPodLogs(clientset, params.Namespace, params.PodName, params.Container)
	if err != nil {
		return nil, err
	}

	return &protocol.CallToolResult{
		Content: []protocol.Content{
			protocol.TextContent{
				Type: "text",
				Text: logs,
			},
		},
	}, nil
}

// Get pod logs
func (p *PodHandler) getPodLogs(clientset kubernetes.Interface, namespace, podName, containerName string) (string, error) {
	// If container name is not specified, try to get Pod info to determine the container
	if containerName == "" {
		pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("error getting pod info: %v", err)
		}

		containers := pod.Spec.Containers
		if len(containers) == 0 {
			return "", fmt.Errorf("no containers found in pod")
		}

		// Default to using the last container
		containerName = containers[len(containers)-1].Name
	}

	logsReq := clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: containerName,
	})
	podLogs, err := logsReq.Stream(context.TODO())
	if err != nil {
		return "", fmt.Errorf("error in opening stream: %v", err)
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", fmt.Errorf("error in copy information from podLogs to buf: %v", err)
	}

	return buf.String(), nil
}

// Delete pod
func (p *PodHandler) delete(_ context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	params, err := biz.ParseParams[deletePodParams](req)
	if err != nil {
		return nil, err
	}

	clientset, err := kubeclient.GetKubeClient()
	if err != nil {
		return nil, err
	}

	err = p.deletePod(clientset, params.Namespace, params.PodName, params.Force)
	return &protocol.CallToolResult{
		Content: []protocol.Content{
			protocol.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Pod %s in namespace %s successfully deleted", params.PodName, params.Namespace),
			},
		},
	}, nil
}

func (p *PodHandler) deletePod(clientset kubernetes.Interface, namespace, podName string, force bool) error {
	deleteOptions := metav1.DeleteOptions{}
	if force {
		gracePeriod := int64(0)
		deleteOptions.GracePeriodSeconds = &gracePeriod
	}
	return clientset.CoreV1().Pods(namespace).Delete(context.TODO(), podName, deleteOptions)
}

// Handle exec_command_in_pod tool
func (p *PodHandler) execCommand(_ context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	params, err := biz.ParseParams[execCommandParams](req)
	if err != nil {
		return nil, err
	}

	// Get clientset
	kubeClient, err := kubeclient.GetKubeClient()
	if err != nil {
		return nil, err
	}

	// Execute command
	output, err := p.execCommandInPod(kubeClient, params.Namespace, params.PodName, params.Command)
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

// Execute command in specified Pod
func (p *PodHandler) execCommandInPod(clientsetInterface kubernetes.Interface, namespace, podName, command string) (string, error) {
	// Create buffers to capture command output
	var stdout, stderr bytes.Buffer

	// Get RESTClient config directly from clientset package
	restConfig, err := kubeclient.GetRESTConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get REST config: %v", err)
	}

	// Build API request
	req := clientsetInterface.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec")

	// Set command parameters
	req.VersionedParams(&corev1.PodExecOptions{
		Command: []string{"/bin/sh", "-c", command},
		Stdin:   false,
		Stdout:  true,
		Stderr:  true,
		TTY:     false,
	}, scheme.ParameterCodec)

	// Create executor
	exec, err := remotecommand.NewSPDYExecutor(restConfig, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("failed to create SPDY executor: %v", err)
	}

	// Execute command and capture output
	err = exec.StreamWithContext(context.TODO(), remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return "", fmt.Errorf("command execution failed: %v, stderr: %s", err, stderr.String())
	}

	// If there is error output, append to standard output
	result := stdout.String()
	if stderr.Len() > 0 {
		if result != "" {
			result += "\n"
		}
		result += "STDERR: " + stderr.String()
	}

	return result, nil
}

// Handle describe_pod tool
func (p *PodHandler) describePod(_ context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	params, err := biz.ParseParams[describePodParams](req)
	if err != nil {
		return nil, err
	}

	// Set default namespace
	if params.Namespace == "" {
		params.Namespace = "default"
	}

	// Get Pod detailed information
	podInfo, err := p.describePodInternal(params.Namespace, params.PodName)
	if err != nil {
		return nil, err
	}

	return &protocol.CallToolResult{
		Content: []protocol.Content{
			protocol.TextContent{
				Type: "text",
				Text: podInfo,
			},
		},
	}, nil
}

// Get detailed Pod information
func (p *PodHandler) describePodInternal(namespace, podName string) (string, error) {
	// Get the latest clientset
	clientset, err := kubeclient.GetKubeClient()
	if err != nil {
		return "", err
	}

	pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get Pod %s info: %v", podName, err)
	}

	var sb strings.Builder

	// Basic information
	sb.WriteString(fmt.Sprintf("Name:\t%s\n", pod.Name))
	sb.WriteString(fmt.Sprintf("Namespace:\t%s\n", pod.Namespace))
	sb.WriteString(fmt.Sprintf("Node:\t%s\n", pod.Spec.NodeName))
	sb.WriteString(fmt.Sprintf("Start Time:\t%s\n", pod.Status.StartTime.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("IP:\t%s\n", pod.Status.PodIP))
	sb.WriteString(fmt.Sprintf("Status:\t%s\n", string(pod.Status.Phase)))

	// Labels
	sb.WriteString("\nLabels:\n")
	for k, v := range pod.Labels {
		sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
	}

	// Container information
	sb.WriteString("\nContainers:\n")
	for _, container := range pod.Spec.Containers {
		sb.WriteString(fmt.Sprintf("  - Name: %s\n", container.Name))
		sb.WriteString(fmt.Sprintf("    Image: %s\n", container.Image))

		// Resource requests and limits
		sb.WriteString("    Resources:\n")
		sb.WriteString("      Requests:\n")
		if cpu, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
			sb.WriteString(fmt.Sprintf("        cpu: %s\n", cpu.String()))
		}
		if memory, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
			sb.WriteString(fmt.Sprintf("        memory: %s\n", memory.String()))
		}

		sb.WriteString("      Limits:\n")
		if cpu, ok := container.Resources.Limits[corev1.ResourceCPU]; ok {
			sb.WriteString(fmt.Sprintf("        cpu: %s\n", cpu.String()))
		}
		if memory, ok := container.Resources.Limits[corev1.ResourceMemory]; ok {
			sb.WriteString(fmt.Sprintf("        memory: %s\n", memory.String()))
		}

		// Ports
		if len(container.Ports) > 0 {
			sb.WriteString("    Ports:\n")
			for _, port := range container.Ports {
				sb.WriteString(fmt.Sprintf("      - %s: %d\n", port.Name, port.ContainerPort))
			}
		}
	}

	// Volumes
	if len(pod.Spec.Volumes) > 0 {
		sb.WriteString("\nVolumes:\n")
		for _, volume := range pod.Spec.Volumes {
			sb.WriteString(fmt.Sprintf("  - Name: %s\n", volume.Name))
			// Add more information based on volume type
			if volume.PersistentVolumeClaim != nil {
				sb.WriteString(fmt.Sprintf("    PVC: %s\n", volume.PersistentVolumeClaim.ClaimName))
			} else if volume.ConfigMap != nil {
				sb.WriteString(fmt.Sprintf("    ConfigMap: %s\n", volume.ConfigMap.Name))
			} else if volume.Secret != nil {
				sb.WriteString(fmt.Sprintf("    Secret: %s\n", volume.Secret.SecretName))
			} else if volume.EmptyDir != nil {
				sb.WriteString("    EmptyDir: {}\n")
			}
		}
	}

	// Events
	events, err := clientset.CoreV1().Events(namespace).List(context.TODO(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.namespace=%s,involvedObject.kind=Pod",
			podName, namespace),
	})

	if err == nil && len(events.Items) > 0 {
		sb.WriteString("\nRecent Events:\n")
		for _, event := range events.Items {
			sb.WriteString(fmt.Sprintf("  %s\t%s\t%s\t%s\n",
				event.LastTimestamp.Format("2006-01-02 15:04:05"),
				event.Type,
				event.Reason,
				event.Message))
		}
	}

	return sb.String(), nil
}

// Handle list_pods tool
func (p *PodHandler) listPods(_ context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var params listPodsParams

	if err := json.Unmarshal(req.RawArguments, &params); err != nil {
		return nil, err
	}

	clientset, err := kubeclient.GetKubeClient()
	if err != nil {
		return nil, err
	}

	// If namespace is not specified and not listing all namespaces, use default
	namespace := params.Namespace
	if namespace == "" && !params.AllNamespaces {
		namespace = "default"
	}

	// Set label selector
	listOptions := metav1.ListOptions{}
	if params.LabelSelector != "" {
		listOptions.LabelSelector = params.LabelSelector
	}

	var pods *corev1.PodList

	if params.AllNamespaces {
		// Get Pods from all namespaces
		pods, err = clientset.CoreV1().Pods("").List(context.TODO(), listOptions)
	} else {
		// Get Pods from specified namespace
		pods, err = clientset.CoreV1().Pods(namespace).List(context.TODO(), listOptions)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get Pod list: %v", err)
	}

	// Format Pod list using formatting utility function
	result := biz.FormatPodsTable(pods.Items)

	return &protocol.CallToolResult{
		Content: []protocol.Content{
			protocol.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}, nil
}
