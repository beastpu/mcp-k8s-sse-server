package node

// NodeParams defines parameters for node operations
type NodeParams struct {
	NodeName string `json:"nodeName"`
}

// NodeListParams defines parameters for listing nodes
type NodeListParams struct {
	LabelSelector string `json:"labelSelector"`
}
