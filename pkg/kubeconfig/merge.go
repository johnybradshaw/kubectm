package kubeconfig

import (
    "fmt"
    "log"
    "os"
    "path/filepath"
    "strings"
    "k8s.io/client-go/tools/clientcmd"
    "k8s.io/client-go/tools/clientcmd/api"
)

// MergeConfigs merges the downloaded kubeconfig files into the main ~/.kube/config
//
// It looks for downloaded kubeconfig files in the ~/.kube directory and merges them
// into the main ~/.kube/config file. After merging, it deletes the individual
// kubeconfig files.
func MergeConfigs() error {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return err
    }

    kubeconfigDir := filepath.Join(homeDir, ".kube")
    mainKubeconfigPath := filepath.Join(kubeconfigDir, "config")

    // Load the main kubeconfig file
    mainConfig, err := loadKubeconfig(mainKubeconfigPath)
    if err != nil {
        log.Printf("No existing kubeconfig found at %s, creating a new one", mainKubeconfigPath)
        mainConfig = api.NewConfig()
    }

    // Find and merge each downloaded kubeconfig file
    files, err := os.ReadDir(kubeconfigDir)
    if err != nil {
        return fmt.Errorf("failed to read kubeconfig directory: %v", err)
    }

    var filesToDelete []string

    for _, file := range files {
        if filepath.Ext(file.Name()) == ".yaml" {
            filePath := filepath.Join(kubeconfigDir, file.Name())
            log.Printf("Merging kubeconfig from %s", filePath)

            newConfig, err := loadKubeconfig(filePath)
            if err != nil {
                return fmt.Errorf("failed to load kubeconfig from %s: %v", filePath, err)
            }

            // Strip the "-kubeconfig" suffix from the file name to get the context name
            contextName := strings.TrimSuffix(file.Name(), "-kubeconfig.yaml")

            err = mergeKubeconfigs(mainConfig, newConfig, contextName)
            if err != nil {
                return fmt.Errorf("failed to merge kubeconfig from %s: %v", filePath, err)
            }

            // Add the file to the list of files to delete after the merge
            filesToDelete = append(filesToDelete, filePath)
        }
    }

    // Save the merged kubeconfig back to the main config file
    err = saveKubeconfig(mainConfig, mainKubeconfigPath)
    if err != nil {
        return fmt.Errorf("failed to save merged kubeconfig: %v", err)
    }

    log.Printf("Successfully merged kubeconfigs into %s", mainKubeconfigPath)

    // Cleanup: delete the individual kubeconfig files after merging
    for _, filePath := range filesToDelete {
        err := os.Remove(filePath)
        if err != nil {
            log.Printf("Warning: failed to delete file %s: %v", filePath, err)
        } else {
            log.Printf("Deleted file %s", filePath)
        }
    }

    return nil
}

// loadKubeconfig loads a kubeconfig file from the specified path
//
// It returns an *api.Config object containing the loaded config and an error
// if any occurred during the loading process.
func loadKubeconfig(path string) (*api.Config, error) {
    // Read the contents of the kubeconfig file at the given path
    kubeconfigBytes, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read kubeconfig file at %s: %v", path, err)
    }

    // Load the bytes into an *api.Config object
    config, err := clientcmd.Load(kubeconfigBytes)
    if err != nil {
        return nil, fmt.Errorf("failed to load kubeconfig file at %s: %v", path, err)
    }

    return config, nil
}

// mergeKubeconfigs merges the source kubeconfig into the destination kubeconfig and renames contexts
//
// This function takes two api.Config objects and a context name, and merges the source
// kubeconfig into the destination. It will rename the context if it already exists, and
// preserves the original cluster name in the context's cluster field. This is done to
// assist the user as providers often use random cluster names.
func mergeKubeconfigs(dest, src *api.Config, contextName string) error {
    // Merge clusters
    for key, cluster := range src.Clusters {
        // Only add the cluster if it doesn't already exist
        if _, exists := dest.Clusters[key]; !exists {
            dest.Clusters[key] = cluster
        }
    }

    // Merge authinfos
    for key, authInfo := range src.AuthInfos {
        // Only add the authInfo if it doesn't already exist
        if _, exists := dest.AuthInfos[key]; !exists {
            dest.AuthInfos[key] = authInfo
        }
    }

    // Merge contexts, renaming if necessary
    for key, context := range src.Contexts {
        // Use the original cluster name in the context's cluster field
        originalClusterName := context.Cluster

        // Ensure the context does not already exist in the destination
        if _, exists := dest.Contexts[contextName]; exists {
            log.Printf("Context %s already exists, skipping...", contextName)
            continue
        }

        // Rename the context to match the desired context name
        uniqueContextName := makeContextNameUnique(contextName, dest.Contexts)

        // Set the new context with the updated name but keep the original cluster reference
        newContext := *context
        newContext.Cluster = originalClusterName
        dest.Contexts[uniqueContextName] = &newContext

        // Update the current context if it was the one being used
        if src.CurrentContext == key {
            dest.CurrentContext = uniqueContextName
        }
    }
    return nil
}

// makeContextNameUnique ensures the context name is unique in the destination contexts
//
// This function takes a context name and a map of existing context names to context objects.
// It checks if the given context name already exists in the map. If it does, it
// appends a number to the end of the name to make it unique (e.g. "mycontext-1").
// It returns the new, unique name.
func makeContextNameUnique(name string, existingContexts map[string]*api.Context) string {
    uniqueName := name
    i := 1
    for {
        if _, exists := existingContexts[uniqueName]; !exists {
            break
        }
        uniqueName = fmt.Sprintf("%s-%d", name, i)
        i++
    }
    return uniqueName
}

// saveKubeconfig saves the merged kubeconfig to the specified path
//
// This function takes an api.Config object and a path where the merged
// kubeconfig should be saved. The merged kubeconfig is written as a YAML
// file to the specified path. If the file already exists, it is overwritten.
// If the path does not include a file name, the file name "config" is used.
// The permissions of the saved file are set to 0600.
func saveKubeconfig(config *api.Config, path string) error {
    // Marshal the api.Config object to YAML
    kubeconfigBytes, err := clientcmd.Write(*config)
    if err != nil {
        // Return an error if marshal fails
        return err
    }

    // Write the YAML to a file at the specified path
    err = os.WriteFile(path, kubeconfigBytes, 0600)
    if err != nil {
        // Return an error if writing the file fails
        return err
    }

    // Return nil if the file is written successfully
    return nil
}
