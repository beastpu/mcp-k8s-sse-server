package configmap

// ConfigMapParams defines parameters for getting a ConfigMap
type ConfigMapParams struct {
	Namespace     string `json:"namespace"`
	ConfigMapName string `json:"configMapName"`
}

// ListConfigMapsParams defines parameters for listing ConfigMaps
type ListConfigMapsParams struct {
	Namespace string `json:"namespace"`
}
