package pod

type podParams struct {
	Namespace string `json:"namespace"`
	PodName   string `json:"podName"`
	Container string `json:"container"`
}

type execCommandParams struct {
	Context   string `json:"context"`
	Namespace string `json:"namespace"`
	PodName   string `json:"podName"`
	Command   string `json:"command"`
}

type describePodParams struct {
	Namespace string `json:"namespace"`
	PodName   string `json:"podName"`
}

type listPodsParams struct {
	Namespace     string `json:"namespace"`
	LabelSelector string `json:"labelSelector"`
	AllNamespaces bool   `json:"allNamespaces"`
}

type deletePodParams struct {
	Namespace string `json:"namespace"`
	PodName   string `json:"podName"`
	Force     bool   `json:"force"`
}
