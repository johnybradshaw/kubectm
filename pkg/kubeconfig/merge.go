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

const (
    errHomeDirFmt = "failed to get user home directory: %v"
)

//go:embed lke.png
var lkeImage []byte

// getKubeDir returns the user's home directory and .kube directory path
func getKubeDir() (homeDir, kubeDir string, err error) {
    homeDir, err = os.UserHomeDir()
    if err != nil {
        return "", "", fmt.Errorf(errHomeDirFmt, err)
    }
    homeDir = filepath.Clean(homeDir)
    kubeDir = filepath.Clean(filepath.Join(homeDir, ".kube"))
    return homeDir, kubeDir, nil
}

// saveImage saves the LKE image to ~/.kube/lke.png
//
// It returns the path to the saved image and an error if saving fails.
func saveImage() (string, error) {
    homeDir, kubeconfigDir, err := getKubeDir()
    if err != nil {
        return "", err
    }

    if !strings.HasPrefix(kubeconfigDir, homeDir) {
        return "", fmt.Errorf("invalid directory path outside user home")
    }

    if err := os.MkdirAll(kubeconfigDir, os.ModePerm); err != nil {
        return "", fmt.Errorf("failed to create kubeconfig directory: %v", err)
    }

    imagePath := filepath.Join(kubeconfigDir, "lke.png")
    if !strings.HasPrefix(imagePath, kubeconfigDir) {
        return "", fmt.Errorf("invalid image path outside .kube directory")
    }

    existingImage, readErr := os.ReadFile(imagePath)
    if readErr != nil || string(existingImage) != string(lkeImage) {
        if err := os.WriteFile(imagePath, lkeImage, 0600); err != nil {
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

// processYAMLFile processes a single YAML kubeconfig file and adds it to the main config
func processYAMLFile(mainConfig *api.Config, kubeconfigDir, fileName, imagePath string) (string, error) {
    filePath := filepath.Clean(filepath.Join(kubeconfigDir, fileName))

    if !strings.HasPrefix(filePath, kubeconfigDir) {
        utils.WarnLogger.Printf("%s Ignoring file outside .kube directory: %s", utils.Iso8601Time(), filePath)
        return "", nil
    }

    utils.ActionLogger.Printf("%s Merging kubeconfig from %s", utils.Iso8601Time(), filePath)

    newConfig, err := loadKubeconfig(filePath)
    if err != nil {
        return "", fmt.Errorf("failed to load kubeconfig from %s: %v", filePath, err)
    }

    contextName := strings.TrimSuffix(fileName, "-kubeconfig.yaml")
    if err := mergeKubeconfigs(mainConfig, newConfig, contextName, imagePath); err != nil {
        return "", fmt.Errorf("failed to merge kubeconfig from %s: %v", filePath, err)
    }

    return filePath, nil
}

// cleanupMergedFiles removes the temporary kubeconfig files that have been merged
func cleanupMergedFiles(filesToDelete []string, kubeconfigDir string) {
    for _, filePath := range filesToDelete {
        if !strings.HasPrefix(filePath, kubeconfigDir) {
            utils.WarnLogger.Printf("%s Skipping deletion of file outside .kube directory: %s", utils.Iso8601Time(), filePath)
            continue
        }

        if err := os.Remove(filePath); err != nil {
            utils.WarnLogger.Printf("%s Warning: failed to delete file %s: %v", utils.Iso8601Time(), filePath, err)
        } else {
            utils.InfoLogger.Printf("%s Deleted file %s", utils.Iso8601Time(), filePath)
        }
    }
}

// MergeConfigs merges all kubeconfig files in the ~/.kube directory into one main config file.
// It ensures safe path operations and cleans up unnecessary files safely.
func MergeConfigs() error {
    homeDir, kubeconfigDir, err := getKubeDir()
    if err != nil {
        return err
    }

    if !strings.HasPrefix(kubeconfigDir, homeDir) {
        return fmt.Errorf("invalid kubeconfig directory outside user home: %s", kubeconfigDir)
    }

    mainKubeconfigPath := filepath.Clean(filepath.Join(kubeconfigDir, "config"))
    if !strings.HasPrefix(mainKubeconfigPath, kubeconfigDir) {
        return fmt.Errorf("invalid main kubeconfig path outside .kube directory: %s", mainKubeconfigPath)
    }

    mainConfig, err := loadKubeconfig(mainKubeconfigPath)
    if err != nil {
        utils.WarnLogger.Printf("%s No existing kubeconfig found at %s, creating a new one", utils.Iso8601Time(), mainKubeconfigPath)
        mainConfig = api.NewConfig()
    }

    imagePath, err := saveImage()
    if err != nil {
        return fmt.Errorf("failed to save Linode icon: %v", err)
    }

    files, err := os.ReadDir(kubeconfigDir)
    if err != nil {
        return fmt.Errorf("failed to read kubeconfig directory: %v", err)
    }

    var filesToDelete []string
    for _, file := range files {
        if filepath.Ext(file.Name()) != ".yaml" {
            continue
        }
        filePath, err := processYAMLFile(mainConfig, kubeconfigDir, file.Name(), imagePath)
        if err != nil {
            return err
        }
        if filePath != "" {
            filesToDelete = append(filesToDelete, filePath)
        }
    }

    if err := saveKubeconfig(mainConfig, mainKubeconfigPath); err != nil {
        return fmt.Errorf("failed to save merged kubeconfig: %v", err)
    }

    utils.InfoLogger.Printf("%s Successfully merged kubeconfigs into %s", utils.Iso8601Time(), mainKubeconfigPath)
    cleanupMergedFiles(filesToDelete, kubeconfigDir)

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
    _, kubeconfigDir, err := getKubeDir()
    if err != nil {
        return nil, err
    }

    if !strings.HasPrefix(filepath.Clean(path), kubeconfigDir) {
        return nil, fmt.Errorf("path traversal attempt detected: %s", path)
    }

    kubeconfigBytes, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read kubeconfig file at %s: %v", path, err)
    }

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

// mergeMaps copies non-existing entries from source maps to destination maps
func mergeClusters(dest, src map[string]*api.Cluster) {
    for key, cluster := range src {
        if _, exists := dest[key]; !exists {
            dest[key] = cluster
        }
    }
}

func mergeAuthInfos(dest, src map[string]*api.AuthInfo) {
    for key, authInfo := range src {
        if _, exists := dest[key]; !exists {
            dest[key] = authInfo
        }
    }
}

// handleExistingContext checks if an existing context should be skipped or overwritten
// Returns: shouldSkip, shouldOverwrite
func handleExistingContext(dest, src *api.Config, contextName string, context *api.Context) (bool, bool) {
    existingContext, exists := dest.Contexts[contextName]
    if !exists {
        return false, false
    }

    srcCluster := src.Clusters[context.Cluster]
    destCluster := dest.Clusters[existingContext.Cluster]

    if isSameCluster(srcCluster, destCluster) {
        utils.ActionLogger.Printf("%s Context %s already exists for the same cluster, skipping...", utils.Iso8601Time(), color.New(color.Bold).Sprint(contextName))
        return true, false
    }

    utils.ActionLogger.Printf("%s Context %s exists but refers to a different cluster, overwriting...", utils.Iso8601Time(), color.New(color.Bold).Sprint(contextName))
    dest.Clusters[context.Cluster] = src.Clusters[context.Cluster]
    if authInfo, exists := src.AuthInfos[context.AuthInfo]; exists {
        dest.AuthInfos[context.AuthInfo] = authInfo
    }
    return false, true
}

// createContextWithExtension creates a new context with the Aptakube extension
func createContextWithExtension(context *api.Context, imagePath string) *api.Context {
    newContext := *context
    newContext.Extensions = map[string]runtime.Object{
        "aptakube": &AptakubeExtension{IconURL: imagePath},
    }
    return &newContext
}

// mergeKubeconfigs merges the source kubeconfig into the destination kubeconfig and renames contexts
func mergeKubeconfigs(dest, src *api.Config, contextName string, imagePath string) error {
    mergeClusters(dest.Clusters, src.Clusters)
    mergeAuthInfos(dest.AuthInfos, src.AuthInfos)

    for key, context := range src.Contexts {
        shouldSkip, shouldOverwrite := handleExistingContext(dest, src, contextName, context)
        if shouldSkip {
            continue
        }

        uniqueContextName := contextName
        if !shouldOverwrite {
            uniqueContextName = makeContextNameUnique(contextName, dest.Contexts)
        }

        newContext := createContextWithExtension(context, imagePath)
        newContext.Cluster = context.Cluster
        dest.Contexts[uniqueContextName] = newContext

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
