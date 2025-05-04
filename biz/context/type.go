package context

// KubeconfigPathParams defines parameters for setting kubeconfig path
type KubeconfigPathParams struct {
	KubeconfigPath string `json:"kubeconfigPath"`
}

// ContextNameParams defines Context name parameters
type ContextNameParams struct {
	ContextName string `json:"contextName"`
}
