package kubeconfig

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "kubectm/pkg/utils"
    "github.com/fatih/color"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/runtime/schema"
    "k8s.io/client-go/tools/clientcmd"
    "k8s.io/client-go/tools/clientcmd/api"
    _ "embed"
)

//go:embed lke.png
var lkeImage []byte

// saveImage saves the LKE image to ~/.kube/lke.png
//
// It returns the path to the saved image and an error if saving fails.
func saveImage() (string, error) {
    // Get the user's home directory
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return "", fmt.Errorf("failed to get user home directory: %v", err)
    }

    // Ensure the home directory is an absolute path
    homeDir = filepath.Clean(homeDir)

    // Create the ~/.kube directory if it doesn't exist
    kubeconfigDir := filepath.Join(homeDir, ".kube")
    if !strings.HasPrefix(kubeconfigDir, homeDir) {
        return "", fmt.Errorf("invalid directory path outside user home")
    }

    err = os.MkdirAll(kubeconfigDir, os.ModePerm)
    if err != nil {
        return "", fmt.Errorf("failed to create kubeconfig directory: %v", err)
    }

    // Determine the path to the image file
    imagePath := filepath.Join(kubeconfigDir, "lke.png")
    if !strings.HasPrefix(imagePath, kubeconfigDir) {
        return "", fmt.Errorf("invalid image path outside .kube directory")
    }

    // Read the existing image file safely
    existingImage, err := os.ReadFile(imagePath)
    
    // If the image file doesn't exist or is different from the embedded image,
    // overwrite it with the embedded image
    if err != nil || string(existingImage) != string(lkeImage) {
        err = os.WriteFile(imagePath, lkeImage, 0600)
        if err != nil {
            return "", fmt.Errorf("failed to write image file: %v", err)
        }
        utils.InfoLogger.Printf("%s Saved Linode icon to %s", utils.Iso8601Time(), imagePath)
    }

    return imagePath, nil
}

// AptakubeExtension is a custom struct that implements runtime.Object
type AptakubeExtension struct {
    IconURL string `json:"icon-url"`
}

// GetObjectKind is required to implement the runtime.Object interface
func (e *AptakubeExtension) GetObjectKind() schema.ObjectKind {
    return schema.EmptyObjectKind
}

// DeepCopyObject is required to implement the runtime.Object interface
func (e *AptakubeExtension) DeepCopyObject() runtime.Object {
    return &AptakubeExtension{
        IconURL: e.IconURL,
    }
}

// MergeConfigs merges all kubeconfig files in the ~/.kube directory into one main config file.
// It ensures safe path operations and cleans up unnecessary files safely.
func MergeConfigs() error {
    // Get the user's home directory and ensure it's an absolute path
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return fmt.Errorf("failed to get user home directory: %v", err)
    }
    homeDir = filepath.Clean(homeDir)

    // Define the .kube directory and ensure it’s within the user's home
    kubeconfigDir := filepath.Join(homeDir, ".kube")
    kubeconfigDir = filepath.Clean(kubeconfigDir)

    if !strings.HasPrefix(kubeconfigDir, homeDir) {
        return fmt.Errorf("invalid kubeconfig directory outside user home: %s", kubeconfigDir)
    }

    // Main kubeconfig file path
    mainKubeconfigPath := filepath.Join(kubeconfigDir, "config")
    mainKubeconfigPath = filepath.Clean(mainKubeconfigPath)

    if !strings.HasPrefix(mainKubeconfigPath, kubeconfigDir) {
        return fmt.Errorf("invalid main kubeconfig path outside .kube directory: %s", mainKubeconfigPath)
    }

    // Load the existing main kubeconfig file
    mainConfig, err := loadKubeconfig(mainKubeconfigPath)
    if err != nil {
        utils.WarnLogger.Printf("%s No existing kubeconfig found at %s, creating a new one", utils.Iso8601Time(), mainKubeconfigPath)
        mainConfig = api.NewConfig()
    }

    imagePath, err := saveImage()
    if err != nil {
        return fmt.Errorf("failed to save Linode icon: %v", err)
    }

    // Read all files in the .kube directory
    files, err := os.ReadDir(kubeconfigDir)
    if err != nil {
        return fmt.Errorf("failed to read kubeconfig directory: %v", err)
    }

    var filesToDelete []string

    for _, file := range files {
        // Only consider files with the .yaml extension
        if filepath.Ext(file.Name()) == ".yaml" {
            filePath := filepath.Join(kubeconfigDir, file.Name())
            filePath = filepath.Clean(filePath)

            // Ensure the file path is within the .kube directory
            if !strings.HasPrefix(filePath, kubeconfigDir) {
                utils.WarnLogger.Printf("%s Ignoring file outside .kube directory: %s", utils.Iso8601Time(), filePath)
                continue
            }

            utils.ActionLogger.Printf("%s Merging kubeconfig from %s", utils.Iso8601Time(), filePath)

            newConfig, err := loadKubeconfig(filePath)
            if err != nil {
                return fmt.Errorf("failed to load kubeconfig from %s: %v", filePath, err)
            }

            // Get the context name from the file name
            contextName := strings.TrimSuffix(file.Name(), "-kubeconfig.yaml")

            err = mergeKubeconfigs(mainConfig, newConfig, contextName, imagePath)
            if err != nil {
                return fmt.Errorf("failed to merge kubeconfig from %s: %v", filePath, err)
            }

            // Add the file to the list of files to delete
            filesToDelete = append(filesToDelete, filePath)
        }
    }

    // Save the merged kubeconfig
    err = saveKubeconfig(mainConfig, mainKubeconfigPath)
    if err != nil {
        return fmt.Errorf("failed to save merged kubeconfig: %v", err)
    }

    utils.InfoLogger.Printf("%s Successfully merged kubeconfigs into %s", utils.Iso8601Time(), mainKubeconfigPath)

    // Delete the other kubeconfig files safely
    for _, filePath := range filesToDelete {
        // Ensure the file path is still within the .kube directory
        if !strings.HasPrefix(filePath, kubeconfigDir) {
            utils.WarnLogger.Printf("%s Skipping deletion of file outside .kube directory: %s", utils.Iso8601Time(), filePath)
            continue
        }

        err := os.Remove(filePath)
        if err != nil {
            utils.WarnLogger.Printf("%s Warning: failed to delete file %s: %v", utils.Iso8601Time(), filePath, err)
        } else {
            utils.InfoLogger.Printf("%s Deleted file %s", utils.Iso8601Time(), filePath)
        }
    }

    return nil
}

// loadKubeconfig loads a kubeconfig file from the specified path safely.
//
// It reads the contents of the file at the specified path and uses the
// clientcmd package to parse the contents into an api.Config object.
// Ensures the path is within the expected ~/.kube directory.
//
// The function returns an api.Config object and an error. If the error
// is not nil, the returned config object is nil.
func loadKubeconfig(path string) (*api.Config, error) {
    // Get the user's home directory and ensure it's an absolute path
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return nil, fmt.Errorf("failed to get user home directory: %v", err)
    }
    homeDir = filepath.Clean(homeDir)

    // Ensure the kubeconfig path is inside the expected ~/.kube directory
    kubeconfigDir := filepath.Join(homeDir, ".kube")
    kubeconfigDir = filepath.Clean(kubeconfigDir)

    if !strings.HasPrefix(filepath.Clean(path), kubeconfigDir) {
        return nil, fmt.Errorf("path traversal attempt detected: %s", path)
    }

    kubeconfigBytes, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read kubeconfig file at %s: %v", path, err)
    }

    // Parse the contents of the file into an api.Config object
    config, err := clientcmd.Load(kubeconfigBytes)
    if err != nil {
        return nil, fmt.Errorf("failed to load kubeconfig file at %s: %v", path, err)
    }

    return config, nil
}

// isSameCluster compares two clusters to determine if they refer to the same Kubernetes instance.
// It compares the server URL and certificate authority data.
func isSameCluster(cluster1, cluster2 *api.Cluster) bool {
    if cluster1 == nil || cluster2 == nil {
        return false
    }

    // Compare server URLs
    if cluster1.Server != cluster2.Server {
        return false
    }

    // Compare certificate authority data
    if string(cluster1.CertificateAuthorityData) != string(cluster2.CertificateAuthorityData) {
        return false
    }

    return true
}

// mergeKubeconfigs merges the source kubeconfig into the destination kubeconfig and renames contexts
func mergeKubeconfigs(dest, src *api.Config, contextName string, imagePath string) error {
    for key, cluster := range src.Clusters {
        if _, exists := dest.Clusters[key]; !exists {
            dest.Clusters[key] = cluster
        }
    }

    for key, authInfo := range src.AuthInfos {
        if _, exists := dest.AuthInfos[key]; !exists {
            dest.AuthInfos[key] = authInfo
        }
    }

    for key, context := range src.Contexts {
        originalClusterName := context.Cluster

        if existingContext, exists := dest.Contexts[contextName]; exists {
            // Check if the existing context refers to the same Kubernetes instance
            srcCluster := src.Clusters[context.Cluster]
            destCluster := dest.Clusters[existingContext.Cluster]

            if isSameCluster(srcCluster, destCluster) {
                utils.ActionLogger.Printf("%s Context %s already exists for the same cluster, skipping...", utils.Iso8601Time(), color.New(color.Bold).Sprint(contextName))
                continue
            }

            // Different cluster with same name - overwrite the existing context
            utils.ActionLogger.Printf("%s Context %s exists but refers to a different cluster, overwriting...", utils.Iso8601Time(), color.New(color.Bold).Sprint(contextName))

            // Update the cluster and auth info
            dest.Clusters[originalClusterName] = src.Clusters[context.Cluster]
            if authInfo, exists := src.AuthInfos[context.AuthInfo]; exists {
                dest.AuthInfos[context.AuthInfo] = authInfo
            }
        }

        uniqueContextName := makeContextNameUnique(contextName, dest.Contexts)

        newContext := *context
        newContext.Cluster = originalClusterName

        // Add the extensions stanza for Linode contexts
        newContext.Extensions = map[string]runtime.Object{
            "aptakube": &AptakubeExtension{
                IconURL: imagePath,
            },
        }

        dest.Contexts[uniqueContextName] = &newContext

        if src.CurrentContext == key {
            dest.CurrentContext = uniqueContextName
        }
    }
    return nil
}

// makeContextNameUnique ensures the context name is unique in the destination contexts
//
// It takes two arguments: the name of the context to be added, and the existing
// contexts in the destination config.
//
// It returns a string representing the context name, possibly modified to be
// unique.
func makeContextNameUnique(name string, existingContexts map[string]*api.Context) string {
    // Start with the original name
    uniqueName := name

    // Loop until we find a name that doesn't exist in the existing contexts
    i := 1
    for {
        // If the name is not in the existing contexts, break out of the loop
        if _, exists := existingContexts[uniqueName]; !exists {
            break
        }

        // Otherwise, increment the counter and try the new name
        uniqueName = fmt.Sprintf("%s-%d", name, i)
        i++
    }

    return uniqueName
}

// saveKubeconfig saves the merged kubeconfig to the specified path
//
// It takes a single argument, a pointer to an api.Config object, which is the
// merged kubeconfig.
//
// The function returns an error if there is a problem writing the file.
func saveKubeconfig(config *api.Config, path string) error {
    // Convert the config object to a byte slice
    kubeconfigBytes, err := clientcmd.Write(*config)
    if err != nil {
        return err
    }

    // Write the byte slice to the specified file path
    err = os.WriteFile(path, kubeconfigBytes, 0600)
    if err != nil {
        return err
    }

    // Return success if the file was written correctly
    return nil
}
