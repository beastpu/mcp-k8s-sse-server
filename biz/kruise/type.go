package kruise

// KruiseNamespaceParams defines namespace related parameters
type KruiseNamespaceParams struct {
	Namespace     string `json:"namespace"`
	AllNamespaces bool   `json:"allNamespaces"`
}

// KruiseScaleParams defines resource scaling parameters
type KruiseScaleParams struct {
	ResourceType string `json:"resourceType"`
	Namespace    string `json:"namespace"`
	ResourceName string `json:"resourceName"`
	Replicas     string `json:"replicas"`
}

// KruiseResourceParams defines resource operation parameters
type KruiseResourceParams struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// KruiseDescribeParams defines parameters for resource description
type KruiseDescribeParams struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}
