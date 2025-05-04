# mcp-k8s

## Overview
mcp-k8s is a lightweight Kubernetes management tool providing both CLI and API interfaces for Kubernetes cluster management. It simplifies the operation of Kubernetes resources, with special support for OpenKruise extensions (such as CloneSet and AdvancedStatefulSet).

## Key Features
- Kubernetes cluster connection and management
- Kubernetes node management (view, cordon, uncordon, restart)
- Pod management (view, delete, log retrieval, command execution)
- OpenKruise resource management (view, describe, and scale CloneSets and AdvancedStatefulSets)
- ConfigMap management
- Multi-cluster context switching

## Requirements
- Go 1.18+
- Kubernetes cluster
- Valid kubeconfig file

## Build Instructions
In the project root directory, run:
```bash
go build -o k8s
```

## Usage
The application supports two running modes:
- stdio mode: For command-line interaction
- SSE mode: Starts an HTTP server for API access

Start in SSE mode:
```bash
./k8s -mode=sse -address=:8686
```

Start in stdio mode:
```bash
./k8s -mode=stdio
```

## Project Structure
- `biz/`: Business logic code
  - `clientset/`: Kubernetes client related code
  - `pod/`: Pod operations
  - `node/`: Node management
  - `context/`: Cluster context management
  - `kruise/`: OpenKruise resource management
  - `configmap/`: ConfigMap management
- `main.go`: Application entry point

## Cursor msp.json
```
{
  "mcpServers": {
    "k8s": {
     "url": "http://127.0.0.1:8686/sse"
    }
  }
}
```

