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
func saveImage() (string, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return "", err
    }

    kubeconfigDir := filepath.Join(homeDir, ".kube")
    err = os.MkdirAll(kubeconfigDir, os.ModePerm)
    if err != nil {
        return "", err
    }

    imagePath := filepath.Join(kubeconfigDir, "lke.png")

    existingImage, err := os.ReadFile(imagePath)
    if err != nil || string(existingImage) != string(lkeImage) {
        err = os.WriteFile(imagePath, lkeImage, 0600)
        if err != nil {
            return "", err
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

func MergeConfigs() error {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return err
    }

    kubeconfigDir := filepath.Join(homeDir, ".kube")
    mainKubeconfigPath := filepath.Join(kubeconfigDir, "config")

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
        if filepath.Ext(file.Name()) == ".yaml" {
            filePath := filepath.Join(kubeconfigDir, file.Name())
            utils.ActionLogger.Printf("%s Merging kubeconfig from %s", utils.Iso8601Time(), filePath)

            newConfig, err := loadKubeconfig(filePath)
            if err != nil {
                return fmt.Errorf("failed to load kubeconfig from %s: %v", filePath, err)
            }

            contextName := strings.TrimSuffix(file.Name(), "-kubeconfig.yaml")

            err = mergeKubeconfigs(mainConfig, newConfig, contextName, imagePath)
            if err != nil {
                return fmt.Errorf("failed to merge kubeconfig from %s: %v", filePath, err)
            }

            filesToDelete = append(filesToDelete, filePath)
        }
    }

    err = saveKubeconfig(mainConfig, mainKubeconfigPath)
    if err != nil {
        return fmt.Errorf("failed to save merged kubeconfig: %v", err)
    }

    utils.InfoLogger.Printf("%s Successfully merged kubeconfigs into %s", utils.Iso8601Time(), mainKubeconfigPath)

    for _, filePath := range filesToDelete {
        err := os.Remove(filePath)
        if err != nil {
            utils.WarnLogger.Printf("%s Warning: failed to delete file %s: %v", utils.Iso8601Time(), filePath, err)
        } else {
            utils.InfoLogger.Printf("%s Deleted file %s", utils.Iso8601Time(), filePath)
        }
    }

    return nil
}

// loadKubeconfig loads a kubeconfig file from the specified path
func loadKubeconfig(path string) (*api.Config, error) {
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

        if _, exists := dest.Contexts[contextName]; exists {
            utils.ActionLogger.Printf("%s %s Context %s already exists, skipping...", utils.Iso8601Time(), color.YellowString(""), color.New(color.Bold).Sprint(contextName))
            continue
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
func saveKubeconfig(config *api.Config, path string) error {
    kubeconfigBytes, err := clientcmd.Write(*config)
    if err != nil {
        return err
    }

    err = os.WriteFile(path, kubeconfigBytes, 0600)
    if err != nil {
        return err
    }

    return nil
}