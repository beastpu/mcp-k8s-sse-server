package biz

import (
	"fmt"
	"strings"
	"time"

	appsv1alpha1 "github.com/openkruise/kruise-api/apps/v1alpha1"
	appsv1beta1 "github.com/openkruise/kruise-api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

// FormatNodesTable formats a list of Nodes as a table string
func FormatNodesTable(nodes []corev1.Node) string {
	if len(nodes) == 0 {
		return "No resources found"
	}

	var sb strings.Builder
	sb.WriteString("NAME\tSTATUS\tROLES\tAGE\tVERSION\n")

	for _, node := range nodes {
		// Get node status
		status := GetNodeStatus(&node)

		// Get node roles
		role := GetNodeRole(&node)

		// Calculate age
		age := CalculateAge(node.CreationTimestamp.Time)

		// Get kubelet version
		version := node.Status.NodeInfo.KubeletVersion

		// Format the row similar to kubectl get nodes output
		sb.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\n",
			node.Name,
			status,
			role,
			age,
			version))
	}

	return sb.String()
}

// FormatConfigMapsTable formats a list of ConfigMaps as a table string
func FormatConfigMapsTable(cms []corev1.ConfigMap) string {
	if len(cms) == 0 {
		return "No resources found"
	}

	var sb strings.Builder
	sb.WriteString("NAMESPACE\tNAME\tDATA\tCREATED AT\n")

	for _, cm := range cms {
		sb.WriteString(fmt.Sprintf("%s\t%s\t%d\t%s\n",
			cm.Namespace,
			cm.Name,
			len(cm.Data),
			cm.CreationTimestamp.Format("2006-01-02 15:04:05")))
	}

	return sb.String()
}

// FormatConfigMapDetail formats a single ConfigMap's detailed view
func FormatConfigMapDetail(cm *corev1.ConfigMap) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("ConfigMap %s details in namespace %s:\n", cm.Name, cm.Namespace))
	sb.WriteString(fmt.Sprintf("Creation Time: %s\n", cm.CreationTimestamp.Format("2006-01-02T15:04:05Z")))

	// Add labels if any
	sb.WriteString("\nLabels:")
	if len(cm.Labels) == 0 {
		sb.WriteString(" <none>\n")
	} else {
		sb.WriteString("\n")
		for k, v := range cm.Labels {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	}

	// Add annotations if any
	sb.WriteString("\nAnnotations:")
	if len(cm.Annotations) == 0 {
		sb.WriteString(" <none>\n")
	} else {
		sb.WriteString("\n")
		for k, v := range cm.Annotations {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	}

	// ConfigMap data
	sb.WriteString("\nData:")
	if len(cm.Data) == 0 {
		sb.WriteString(" <none>\n")
	} else {
		sb.WriteString("\n")
		for k, v := range cm.Data {
			sb.WriteString(fmt.Sprintf("---\nKey: %s\nValue:\n%s\n---\n", k, v))
		}
	}

	// ConfigMap binary data if present
	if len(cm.BinaryData) > 0 {
		sb.WriteString("\nBinary Data:\n")
		for k := range cm.BinaryData {
			sb.WriteString(fmt.Sprintf("  %s: <binary data>\n", k))
		}
	}

	return sb.String()
}

// FormatPodsTable formats a list of Pods as a table string
func FormatPodsTable(pods []corev1.Pod) string {
	if len(pods) == 0 {
		return "No resources found"
	}

	var sb strings.Builder
	sb.WriteString("NAMESPACE\tNAME\tSTATUS\tSTART TIME\tIP\n")

	for _, pod := range pods {
		// Get detailed Pod status
		status := GetPodStatus(&pod)

		sb.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\n",
			pod.Namespace,
			pod.Name,
			status,
			pod.Status.StartTime.Format("2006-01-02 15:04:05"),
			pod.Status.PodIP))
	}

	return sb.String()
}

// GetPodStatus gets the detailed Pod status, consistent with kubectl get pods output
func GetPodStatus(pod *corev1.Pod) string {
	if pod.DeletionTimestamp != nil {
		return "Terminating"
	}

	// If Pod is in Failed or Succeeded phase, return directly
	if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
		return string(pod.Status.Phase)
	}

	// Handle Running Pods special status
	if pod.Status.Phase == corev1.PodRunning {
		// Check if all containers are ready
		allContainersReady := true
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if !containerStatus.Ready {
				allContainersReady = false
				break
			}
		}
		if allContainersReady {
			return string(pod.Status.Phase)
		}
	}

	// Handle special statuses
	// First check init container status
	for i, status := range pod.Status.InitContainerStatuses {
		if status.State.Waiting != nil && status.State.Waiting.Reason != "" {
			return fmt.Sprintf("Init:%s", status.State.Waiting.Reason)
		}
		if status.State.Terminated != nil && status.State.Terminated.Reason != "" {
			return fmt.Sprintf("Init:%s", status.State.Terminated.Reason)
		}
		// Initializing, display the number of completed init containers
		if status.State.Terminated == nil || status.State.Terminated.Reason == "" {
			return fmt.Sprintf("Init:%d/%d", i, len(pod.Spec.InitContainers))
		}
	}

	// Handle normal container status
	hasRunning := false
	waiting := false
	waitingReason := ""
	terminated := false
	terminatedReason := ""

	for _, status := range pod.Status.ContainerStatuses {
		if status.State.Running != nil {
			hasRunning = true
		}
		if status.State.Waiting != nil && status.State.Waiting.Reason != "" {
			waiting = true
			waitingReason = status.State.Waiting.Reason
		}
		if status.State.Terminated != nil && status.State.Terminated.Reason != "" {
			terminated = true
			terminatedReason = status.State.Terminated.Reason
		}
	}

	// Prioritize returning the waiting status reason, such as ImagePullBackOff, CrashLoopBackOff, etc.
	if waiting {
		return waitingReason
	} else if terminated {
		return terminatedReason
	} else if hasRunning {
		// If containers are running but not all are ready, display as NotReady
		if pod.Status.Phase == corev1.PodRunning {
			return "Running,NotReady"
		}
	}

	// By default, return the Pod's own phase
	return string(pod.Status.Phase)
}

// FormatAdvancedStatefulSetsTable formats a list of AdvancedStatefulSets as a table string
func FormatAdvancedStatefulSetsTable(items []appsv1beta1.StatefulSet) string {
	if len(items) == 0 {
		return "No resources found"
	}

	var sb strings.Builder
	sb.WriteString("NAMESPACE\tNAME\tREPLICAS\tREADY\tUPDATED\tAGE\n")

	for _, ast := range items {
		// Calculate age
		age := CalculateAge(ast.CreationTimestamp.Time)

		// Get replicas info
		var replicas int32 = 0
		if ast.Spec.Replicas != nil {
			replicas = *ast.Spec.Replicas
		}

		// Format the row
		sb.WriteString(fmt.Sprintf("%s\t%s\t%d/%d\t%d\t%d\t%s\n",
			ast.Namespace,
			ast.Name,
			ast.Status.ReadyReplicas,
			replicas,
			ast.Status.ReadyReplicas,
			ast.Status.UpdatedReplicas,
			age))
	}

	return sb.String()
}

// FormatCloneSetsTable formats a list of CloneSets as a table string
func FormatCloneSetsTable(items []appsv1alpha1.CloneSet) string {
	if len(items) == 0 {
		return "No resources found"
	}

	var sb strings.Builder
	sb.WriteString("NAMESPACE\tNAME\tREPLICAS\tREADY\tUPDATED\tAGE\n")

	for _, cs := range items {
		// Calculate age
		age := CalculateAge(cs.CreationTimestamp.Time)

		// Get replicas info
		var replicas int32 = 0
		if cs.Spec.Replicas != nil {
			replicas = *cs.Spec.Replicas
		}

		// Format the row
		sb.WriteString(fmt.Sprintf("%s\t%s\t%d/%d\t%d\t%d\t%s\n",
			cs.Namespace,
			cs.Name,
			cs.Status.ReadyReplicas,
			replicas,
			cs.Status.ReadyReplicas,
			cs.Status.UpdatedReplicas,
			age))
	}

	return sb.String()
}

// FormatAdvancedStatefulSetDetail formats a single AdvancedStatefulSet detailed description
func FormatAdvancedStatefulSetDetail(ast *appsv1beta1.StatefulSet) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Name:               %s\n", ast.Name))
	sb.WriteString(fmt.Sprintf("Namespace:          %s\n", ast.Namespace))

	// Format age
	age := CalculateAge(ast.CreationTimestamp.Time)
	sb.WriteString(fmt.Sprintf("CreationTimestamp:  %s (%s ago)\n",
		ast.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
		age))

	// Replicas info
	var replicas int32 = 0
	if ast.Spec.Replicas != nil {
		replicas = *ast.Spec.Replicas
	}
	sb.WriteString(fmt.Sprintf("Replicas:           %d\n", replicas))
	sb.WriteString(fmt.Sprintf("Status Replicas:    %d\n", ast.Status.Replicas))
	sb.WriteString(fmt.Sprintf("Ready Replicas:     %d\n", ast.Status.ReadyReplicas))
	sb.WriteString(fmt.Sprintf("Current Replicas:   %d\n", ast.Status.CurrentReplicas))
	sb.WriteString(fmt.Sprintf("Updated Replicas:   %d\n", ast.Status.UpdatedReplicas))

	// Pod Management Policy
	sb.WriteString(fmt.Sprintf("Pod Management Policy: %s\n", ast.Spec.PodManagementPolicy))

	// Update Strategy
	sb.WriteString(fmt.Sprintf("Update Strategy:    %s\n", ast.Spec.UpdateStrategy.Type))
	if ast.Spec.UpdateStrategy.RollingUpdate != nil {
		if ast.Spec.UpdateStrategy.RollingUpdate.Partition != nil {
			sb.WriteString(fmt.Sprintf("  Partition:       %d\n", *ast.Spec.UpdateStrategy.RollingUpdate.Partition))
		}
		if ast.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable != nil {
			sb.WriteString(fmt.Sprintf("  MaxUnavailable:  %s\n", ast.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable.String()))
		}
	}

	// Selector
	sb.WriteString("Selector:\n")
	if ast.Spec.Selector != nil && len(ast.Spec.Selector.MatchLabels) > 0 {
		for k, v := range ast.Spec.Selector.MatchLabels {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	} else {
		sb.WriteString("  <none>\n")
	}

	// Labels
	sb.WriteString("Labels:")
	if len(ast.Labels) == 0 {
		sb.WriteString(" <none>\n")
	} else {
		sb.WriteString("\n")
		for k, v := range ast.Labels {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	}

	// Annotations
	sb.WriteString("Annotations:")
	if len(ast.Annotations) == 0 {
		sb.WriteString(" <none>\n")
	} else {
		sb.WriteString("\n")
		for k, v := range ast.Annotations {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	}

	// Pod Template
	sb.WriteString("Pod Template:\n")
	sb.WriteString("  Labels:")
	if len(ast.Spec.Template.Labels) == 0 {
		sb.WriteString(" <none>\n")
	} else {
		sb.WriteString("\n")
		for k, v := range ast.Spec.Template.Labels {
			sb.WriteString(fmt.Sprintf("    %s: %s\n", k, v))
		}
	}

	// Containers
	sb.WriteString("  Containers:\n")
	if len(ast.Spec.Template.Spec.Containers) == 0 {
		sb.WriteString("    <none>\n")
	} else {
		for _, container := range ast.Spec.Template.Spec.Containers {
			sb.WriteString(fmt.Sprintf("   - Name:  %s\n", container.Name))
			sb.WriteString(fmt.Sprintf("     Image: %s\n", container.Image))

			// Resource requirements
			if container.Resources.Limits != nil || container.Resources.Requests != nil {
				sb.WriteString("     Resources:\n")
				if container.Resources.Limits != nil && len(container.Resources.Limits) > 0 {
					sb.WriteString("       Limits:\n")
					for resName, quantity := range container.Resources.Limits {
						sb.WriteString(fmt.Sprintf("         %s: %s\n", resName, quantity.String()))
					}
				}
				if container.Resources.Requests != nil && len(container.Resources.Requests) > 0 {
					sb.WriteString("       Requests:\n")
					for resName, quantity := range container.Resources.Requests {
						sb.WriteString(fmt.Sprintf("         %s: %s\n", resName, quantity.String()))
					}
				}
			}
		}
	}

	// Volume Claims
	if len(ast.Spec.VolumeClaimTemplates) > 0 {
		sb.WriteString("Volume Claim Templates:\n")
		for _, pvc := range ast.Spec.VolumeClaimTemplates {
			sb.WriteString(fmt.Sprintf("  %s:\n", pvc.Name))
			sb.WriteString(fmt.Sprintf("    AccessModes: %v\n", pvc.Spec.AccessModes))
			if pvc.Spec.Resources.Requests != nil {
				storage, exists := pvc.Spec.Resources.Requests["storage"]
				if exists {
					sb.WriteString(fmt.Sprintf("    Storage Request: %s\n", storage.String()))
				}
			}
			if pvc.Spec.StorageClassName != nil {
				sb.WriteString(fmt.Sprintf("    StorageClass: %s\n", *pvc.Spec.StorageClassName))
			}
		}
	}

	return sb.String()
}

// FormatCloneSetDetail formats a single CloneSet detailed description
func FormatCloneSetDetail(cloneSet *appsv1alpha1.CloneSet) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Name:               %s\n", cloneSet.Name))
	sb.WriteString(fmt.Sprintf("Namespace:          %s\n", cloneSet.Namespace))

	// Format age
	age := CalculateAge(cloneSet.CreationTimestamp.Time)
	sb.WriteString(fmt.Sprintf("CreationTimestamp:  %s (%s ago)\n",
		cloneSet.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
		age))

	// Replicas info
	var replicas int32 = 0
	if cloneSet.Spec.Replicas != nil {
		replicas = *cloneSet.Spec.Replicas
	}
	sb.WriteString(fmt.Sprintf("Replicas:           %d\n", replicas))
	sb.WriteString(fmt.Sprintf("Status Replicas:    %d\n", cloneSet.Status.Replicas))
	sb.WriteString(fmt.Sprintf("Ready Replicas:     %d\n", cloneSet.Status.ReadyReplicas))
	sb.WriteString(fmt.Sprintf("Available Replicas: %d\n", cloneSet.Status.AvailableReplicas))
	sb.WriteString(fmt.Sprintf("Updated Replicas:   %d\n", cloneSet.Status.UpdatedReplicas))
	sb.WriteString(fmt.Sprintf("Updated Ready Replicas: %d\n", cloneSet.Status.UpdatedReadyReplicas))

	// Update Strategy
	sb.WriteString(fmt.Sprintf("Update Strategy:    %s\n", cloneSet.Spec.UpdateStrategy.Type))
	if cloneSet.Spec.UpdateStrategy.Partition != nil {
		sb.WriteString(fmt.Sprintf("  Partition:       %s\n", cloneSet.Spec.UpdateStrategy.Partition.String()))
	}
	if cloneSet.Spec.UpdateStrategy.MaxUnavailable != nil {
		sb.WriteString(fmt.Sprintf("  MaxUnavailable:  %s\n", cloneSet.Spec.UpdateStrategy.MaxUnavailable.String()))
	}
	if cloneSet.Spec.UpdateStrategy.MaxSurge != nil {
		sb.WriteString(fmt.Sprintf("  MaxSurge:        %s\n", cloneSet.Spec.UpdateStrategy.MaxSurge.String()))
	}

	// Selector
	sb.WriteString("Selector:\n")
	if cloneSet.Spec.Selector != nil && len(cloneSet.Spec.Selector.MatchLabels) > 0 {
		for k, v := range cloneSet.Spec.Selector.MatchLabels {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	} else {
		sb.WriteString("  <none>\n")
	}

	// Labels
	sb.WriteString("Labels:")
	if len(cloneSet.Labels) == 0 {
		sb.WriteString(" <none>\n")
	} else {
		sb.WriteString("\n")
		for k, v := range cloneSet.Labels {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	}

	// Annotations
	sb.WriteString("Annotations:")
	if len(cloneSet.Annotations) == 0 {
		sb.WriteString(" <none>\n")
	} else {
		sb.WriteString("\n")
		for k, v := range cloneSet.Annotations {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	}

	// Pod Template
	sb.WriteString("Pod Template:\n")
	sb.WriteString("  Labels:")
	if len(cloneSet.Spec.Template.Labels) == 0 {
		sb.WriteString(" <none>\n")
	} else {
		sb.WriteString("\n")
		for k, v := range cloneSet.Spec.Template.Labels {
			sb.WriteString(fmt.Sprintf("    %s: %s\n", k, v))
		}
	}

	// Containers
	sb.WriteString("  Containers:\n")
	if len(cloneSet.Spec.Template.Spec.Containers) == 0 {
		sb.WriteString("    <none>\n")
	} else {
		for _, container := range cloneSet.Spec.Template.Spec.Containers {
			sb.WriteString(fmt.Sprintf("   - Name:  %s\n", container.Name))
			sb.WriteString(fmt.Sprintf("     Image: %s\n", container.Image))

			// Resource requirements
			if container.Resources.Limits != nil || container.Resources.Requests != nil {
				sb.WriteString("     Resources:\n")
				if container.Resources.Limits != nil && len(container.Resources.Limits) > 0 {
					sb.WriteString("       Limits:\n")
					for resName, quantity := range container.Resources.Limits {
						sb.WriteString(fmt.Sprintf("         %s: %s\n", resName, quantity.String()))
					}
				}
				if container.Resources.Requests != nil && len(container.Resources.Requests) > 0 {
					sb.WriteString("       Requests:\n")
					for resName, quantity := range container.Resources.Requests {
						sb.WriteString(fmt.Sprintf("         %s: %s\n", resName, quantity.String()))
					}
				}
			}
		}
	}

	return sb.String()
}

// CalculateAge calculates a human-readable age string from creation time
func CalculateAge(creationTime time.Time) string {
	duration := time.Since(creationTime)
	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd", days)
	} else if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	} else {
		return fmt.Sprintf("%dm", minutes)
	}
}

// FormatNodeInfoTable formats detailed Node information in kubectl describe style
func FormatNodeInfoTable(node *corev1.Node) string {
	var sb strings.Builder

	// Basic information
	sb.WriteString(fmt.Sprintf("Name:                 %s\n", node.Name))
	role := GetNodeRole(node)
	sb.WriteString(fmt.Sprintf("Role:                 %s\n", role))

	sb.WriteString("Labels:")
	if len(node.Labels) == 0 {
		sb.WriteString(" <none>\n")
	} else {
		sb.WriteString(fmt.Sprintf("               %d\n", len(node.Labels)))
		for k, v := range node.Labels {
			sb.WriteString(fmt.Sprintf("                      %s=%s\n", k, v))
		}
	}

	sb.WriteString("Annotations:")
	if len(node.Annotations) == 0 {
		sb.WriteString(" <none>\n")
	} else {
		sb.WriteString(fmt.Sprintf("          %d\n", len(node.Annotations)))
	}

	age := CalculateAge(node.CreationTimestamp.Time)
	sb.WriteString(fmt.Sprintf("CreationTimestamp:    %s (%s ago)\n",
		node.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
		age))

	// Node status
	status := GetNodeStatus(node)
	sb.WriteString(fmt.Sprintf("Status:               %s\n", status))

	// Node addresses
	sb.WriteString("Addresses:\n")
	if len(node.Status.Addresses) == 0 {
		sb.WriteString("  <none>\n")
	} else {
		for _, addr := range node.Status.Addresses {
			sb.WriteString(fmt.Sprintf("  %s:    %s\n", addr.Type, addr.Address))
		}
	}

	// Node capacity
	sb.WriteString("Capacity:\n")
	if len(node.Status.Capacity) == 0 {
		sb.WriteString("  <none>\n")
	} else {
		for k, v := range node.Status.Capacity {
			sb.WriteString(fmt.Sprintf("  %s:    %s\n", k, v.String()))
		}
	}

	sb.WriteString("Allocatable:\n")
	if len(node.Status.Allocatable) == 0 {
		sb.WriteString("  <none>\n")
	} else {
		for k, v := range node.Status.Allocatable {
			sb.WriteString(fmt.Sprintf("  %s:    %s\n", k, v.String()))
		}
	}

	// System info
	sb.WriteString("System Info:\n")
	sb.WriteString(fmt.Sprintf("  OS Image:                    %s\n", node.Status.NodeInfo.OSImage))
	sb.WriteString(fmt.Sprintf("  Kernel Version:              %s\n", node.Status.NodeInfo.KernelVersion))
	sb.WriteString(fmt.Sprintf("  Container Runtime Version:   %s\n", node.Status.NodeInfo.ContainerRuntimeVersion))
	sb.WriteString(fmt.Sprintf("  Kubelet Version:             %s\n", node.Status.NodeInfo.KubeletVersion))
	sb.WriteString(fmt.Sprintf("  Kube-Proxy Version:          %s\n", node.Status.NodeInfo.KubeProxyVersion))

	return sb.String()
}

// GetNodeStatus returns the status of a Node
func GetNodeStatus(node *corev1.Node) string {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				return "Ready"
			} else {
				return "NotReady"
			}
		}
	}
	return "Unknown"
}

// GetNodeRole extracts the role from Node labels
func GetNodeRole(node *corev1.Node) string {
	roles := []string{}
	for label := range node.Labels {
		if strings.HasPrefix(label, "node-role.kubernetes.io/") {
			role := strings.TrimPrefix(label, "node-role.kubernetes.io/")
			roles = append(roles, role)
		}
	}

	if len(roles) == 0 {
		return "<none>"
	}

	return strings.Join(roles, ",")
}
