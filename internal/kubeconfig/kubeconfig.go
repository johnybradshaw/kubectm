package kubeconfig

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/johnybradshaw/acc-kubeconfig-cli/internal/constants" // Import the constants package
	"gopkg.in/yaml.v2"
)

// CreateKubeDirectory creates the .kube directory in the user's home directory.
//
// This function does not take any parameters.
// It returns an error if there was a problem creating the directory.
func CreateKubeDirectory() {
	err := os.MkdirAll(filepath.Join(os.Getenv("HOME"), ".kube"), os.ModePerm)
	if err != nil {
		fmt.Printf("%sError:%s Failed to create .kube directory: %v\n", constants.ColorRed, constants.ColorReset, err)
		os.Exit(1)
	}
}

// InitializeKubeconfigFile initializes the kubeconfig file if it does not exist.
//
// This function does not take any parameters.
// It returns a string representing the path of the kubeconfig file.
func InitializeKubeconfigFile() string {
	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		err := os.WriteFile(kubeconfigPath, []byte(`apiVersion: v1
kind: Config
clusters: []
contexts: []
users: []
current-context: ""`), os.ModePerm)
		if err != nil {
			fmt.Printf("%sError:%s Failed to create empty kubeconfig file: %v\n", constants.ColorRed, constants.ColorReset, err)
			os.Exit(1)
		}
	}
	return kubeconfigPath
}

// UpdateContextName updates the name of a context in the given kubeconfig file.
//
// Parameters:
// - kubeconfigPath: The path to the kubeconfig file.
// - oldContextName: The current name of the context to be updated.
// - newContextName: The new name to be assigned to the context.
//
// Returns:
// - error: An error if there was a problem reading or writing the kubeconfig file, or if the context was not found.
func UpdateContextName(kubeconfigPath, oldContextName, newContextName string) error {
	data, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return err
	}

	var kubeconfig map[string]interface{}
	err = yaml.Unmarshal(data, &kubeconfig)
	if err != nil {
		return err
	}

	// Update the context name
	contexts, ok := kubeconfig["contexts"].([]interface{})
	if ok {
		for _, context := range contexts {
			contextMap, ok := context.(map[string]interface{})
			if ok {
				if contextMap["name"] == oldContextName {
					contextMap["name"] = newContextName
					break
				}
			}
		}
	}

	updatedData, err := yaml.Marshal(kubeconfig)
	if err != nil {
		return err
	}

	err = os.WriteFile(kubeconfigPath, updatedData, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

// MergeKubeconfigs merges the given kubeconfig with the existing kubeconfig file.
//
// Parameters:
// - kubeconfig: The content of the kubeconfig to be merged.
// - kubeconfigPath: The path to the existing kubeconfig file.
//
// Returns:
// - error: An error if there was a problem merging the kubeconfigs or writing the merged kubeconfig to the file.
func MergeKubeconfigs(kubeconfig string, kubeconfigPath string) error {
	cmd := exec.Command("kubectl", "config", "view", "--flatten")
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeconfig))
	mergedKubeconfig, err := cmd.Output()
	if err != nil {
		return err
	}
	err = os.WriteFile(kubeconfigPath+".tmp", mergedKubeconfig, os.ModePerm)
	if err != nil {
		return err
	}
	err = os.Rename(kubeconfigPath+".tmp", kubeconfigPath)
	if err != nil {
		return err
	}
	return nil
}
