package clientset

import (
	"fmt"
	"os"
	"sync/atomic"

	kruiseclientset "github.com/openkruise/kruise-api/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

var (
	currentContext       string
	customKubeconfigPath string 
	clientsetPointer     atomic.Pointer[kubernetes.Clientset]
	kruiseClientPointer  atomic.Pointer[kruiseclientset.Clientset]
)

// ValidateAndFixKubeconfig validates and fixes invalid kubeconfig files
func ValidateAndFixKubeconfig(path string) error {
	fmt.Printf("Debug - Validating kubeconfig file: %s\n", path)

	// Check if file exists
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("kubeconfig file does not exist: %s", path)
	}

	// Try to read file content
	_, err = os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("unable to read kubeconfig file: %v", err)
	}

	// Try to load the file
	config, err := clientcmd.LoadFromFile(path)
	if err != nil {
		return fmt.Errorf("invalid kubeconfig file format: %v", err)
	}

	// Check if config is a valid clientcmdapi.Config
	if !isValidConfig(config) {
		return fmt.Errorf("invalid kubeconfig configuration, must contain at least one cluster, context, and user")
	}

	fmt.Printf("Debug - Kubeconfig file validation successful, contains %d clusters, %d contexts, %d users\n",
		len(config.Clusters), len(config.Contexts), len(config.AuthInfos))

	return nil
}

// isValidConfig checks if config is a valid clientcmdapi.Config
func isValidConfig(config *clientcmdapi.Config) bool {
	// At least one cluster, context, and user
	return len(config.Clusters) > 0 && len(config.Contexts) > 0 && len(config.AuthInfos) > 0
}

// GetCurrentContext gets the current context
func GetCurrentContext() (string, error) {
	// If current context is already set, return it directly
	if currentContext != "" {
		return currentContext, nil
	}

	// Otherwise, get from kubeconfig file
	config, err := GetKubeConfig()
	if err != nil {
		return "", err
	}

	// Output debug information
	fmt.Printf("Debug - CurrentContext from kubeconfig: %s\n", config.CurrentContext)
	fmt.Printf("Debug - Number of available contexts: %d\n", len(config.Contexts))

	// Save current context
	currentContext = config.CurrentContext

	// If current context is empty but there are available contexts, try to use the first available context
	if currentContext == "" && len(config.Contexts) > 0 {
		// Get the name of the first available context
		for name := range config.Contexts {
			fmt.Printf("Debug - Found available context: %s, setting as current context\n", name)
			currentContext = name
			break
		}
	}

	return currentContext, nil
}

// GetKubeConfig gets kubeconfig configuration
func GetKubeConfig() (*clientcmdapi.Config, error) {
	var configAccess clientcmd.ConfigAccess

	if customKubeconfigPath != "" {
		// Check if custom kubeconfig file exists
		_, err := clientcmd.LoadFromFile(customKubeconfigPath)
		if err != nil {
			fmt.Printf("Debug - Unable to load custom kubeconfig file: %v\n", err)
			return nil, fmt.Errorf("unable to load custom kubeconfig file(%s): %v", customKubeconfigPath, err)
		}
		fmt.Printf("Debug - Using custom kubeconfig path: %s\n", customKubeconfigPath)
		// Use custom kubeconfig path
		configAccess = &clientcmd.ClientConfigLoadingRules{ExplicitPath: customKubeconfigPath}
	} else {
		// Use default kubeconfig path
		fmt.Println("Debug - Using default kubeconfig path")
		configAccess = clientcmd.NewDefaultClientConfigLoadingRules()
	}

	config, err := configAccess.GetStartingConfig()
	if err != nil {
		fmt.Printf("Debug - Failed to get kubeconfig configuration: %v\n", err)
		return nil, fmt.Errorf("error loading kubeconfig: %v", err)
	}
	return config, nil
}

// GetKubeClient gets Kubernetes client
func GetKubeClient() (kubernetes.Interface, error) {
	// First try to load existing clientset from atomic pointer
	if cs := clientsetPointer.Load(); cs != nil {
		return cs, nil
	}

	// If no cached clientset, create a new one
	contextName, err := GetCurrentContext()
	if err != nil {
		return nil, err
	}

	// Create client config loader
	var configLoader clientcmd.ClientConfig

	if customKubeconfigPath != "" {
		// Use custom kubeconfig path
		configLoader = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: customKubeconfigPath},
			&clientcmd.ConfigOverrides{
				CurrentContext: contextName,
			})
	} else {
		// Use default kubeconfig path
		configLoader = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			clientcmd.NewDefaultClientConfigLoadingRules(),
			&clientcmd.ConfigOverrides{
				CurrentContext: contextName,
			})
	}

	config, err := configLoader.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig for context %s: %v", contextName, err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client for context %s: %v", contextName, err)
	}

	// Store newly created clientset in atomic pointer
	clientsetPointer.Store(clientset)

	return clientset, nil
}

// getKruiseClient gets OpenKruise client
func getKruiseClient() (kruiseclientset.Interface, error) {
	// If context name is not specified, use current context
	contextName, err := GetCurrentContext()
	if err != nil {
		return nil, err
	}

	if kc := kruiseClientPointer.Load(); kc != nil {
		return kc, nil
	}

	var configLoader clientcmd.ClientConfig

	if customKubeconfigPath != "" {
		// Use custom kubeconfig path
		configLoader = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: customKubeconfigPath},
			&clientcmd.ConfigOverrides{
				CurrentContext: contextName,
			})
	} else {
		// Use default kubeconfig path
		configLoader = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			clientcmd.NewDefaultClientConfigLoadingRules(),
			&clientcmd.ConfigOverrides{
				CurrentContext: contextName,
			})
	}

	config, err := configLoader.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig for context %s: %v", contextName, err)
	}

	// Create OpenKruise client
	kruiseClient, err := kruiseclientset.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenKruise client for context %s: %v", contextName, err)
	}

	// Store newly created kruise client in atomic pointer
	kruiseClientPointer.Store(kruiseClient)

	return kruiseClient, nil
}

// GetKruiseClient exported OpenKruise client getter function for use by sub-packages
func GetKruiseClient() (kruiseclientset.Interface, error) {
	return getKruiseClient()
}

// ClearClientCache clears client cache
func ClearClientCache() {
	// Clear clientset and kruiseClient cache
	clientsetPointer.Store(nil)
	kruiseClientPointer.Store(nil)
	fmt.Println("Debug - Client cache cleared")
}

// SetCustomKubeconfigPath sets custom kubeconfig path
func SetCustomKubeconfigPath(path string) {
	customKubeconfigPath = path
}

// GetCustomKubeconfigPath gets custom kubeconfig path
func GetCustomKubeconfigPath() string {
	return customKubeconfigPath
}

// ResetCurrentContext resets current context
func ResetCurrentContext() {
	currentContext = ""
}

// SetCurrentContext sets current context
func SetCurrentContext(ctx string) {
	currentContext = ctx
}

// GetRESTConfig gets configuration for creating REST client
func GetRESTConfig() (*rest.Config, error) {
	// Get current context
	contextName, err := GetCurrentContext()
	if err != nil {
		return nil, err
	}

	var configLoader clientcmd.ClientConfig

	if customKubeconfigPath != "" {
		// Use custom kubeconfig path
		configLoader = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: customKubeconfigPath},
			&clientcmd.ConfigOverrides{
				CurrentContext: contextName,
			})
	} else {
		// Use default kubeconfig path
		configLoader = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			clientcmd.NewDefaultClientConfigLoadingRules(),
			&clientcmd.ConfigOverrides{
				CurrentContext: contextName,
			})
	}

	// Load kubeconfig file
	config, err := configLoader.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig for context %s: %v", contextName, err)
	}

	return config, nil
}
