package kubeconfig

import (
    "fmt"
    "os"
    "os/exec"
    "path/filepath"

    "github.com/johnybradshaw/acc-kubeconfig-cli/internal/constants" // Import the constants package
    "gopkg.in/yaml.v2"
)

func CreateKubeDirectory() {
	err := os.MkdirAll(filepath.Join(os.Getenv("HOME"), ".kube"), os.ModePerm)
	if err != nil {
		fmt.Printf("%sError:%s Failed to create .kube directory: %v\n", constants.ColorRed, constants.ColorReset, err)
		os.Exit(1)
	}
}

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
