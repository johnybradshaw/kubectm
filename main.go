package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/linode/linodego"
	"github.com/spf13/pflag"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
)

var (
	debugMode bool
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

func init() {
	pflag.BoolVar(&debugMode, "debug", false, "Enable debug mode to print additional information during script execution")
	pflag.Parse()
}

func main() {
	// Check if required dependencies are installed
	dependencies := []string{"kubectl"}
	for _, dependency := range dependencies {
		_, err := exec.LookPath(dependency)
		if err != nil {
			fmt.Printf("%sError:%s %s is not installed. Please install it and try again.\n", colorRed, colorReset, dependency)
			os.Exit(1)
		}
	}

	// Create the .kube directory if it doesn't exist
	err := os.MkdirAll(filepath.Join(os.Getenv("HOME"), ".kube"), os.ModePerm)
	if err != nil {
		fmt.Printf("%sError:%s Failed to create .kube directory: %v\n", colorRed, colorReset, err)
		os.Exit(1)
	}

	// Initialize an empty kubeconfig file if it doesn't exist
	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		err := os.WriteFile(kubeconfigPath, []byte(`apiVersion: v1
kind: Config
clusters: []
contexts: []
users: []
current-context: ""`), os.ModePerm)
		if err != nil {
			fmt.Printf("%sError:%s Failed to create empty kubeconfig file: %v\n", colorRed, colorReset, err)
			os.Exit(1)
		}
	}

	// Get the Linode API token from the environment variable
	apiToken := os.Getenv("LINODE_API_TOKEN")
	if apiToken == "" {
		fmt.Printf("%sError:%s %sLINODE_API_TOKEN%s environment variable is not set\n", colorRed, colorReset, colorCyan, colorReset)
		os.Exit(1)
	}

	// Create a Linode client
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: apiToken})
	oauth2Client := &http.Client{
		Transport: &oauth2.Transport{
			Source: tokenSource,
		},
	}
	linodeClient := linodego.NewClient(oauth2Client)

	// Get the list of cluster IDs
	clusters, err := linodeClient.ListLKEClusters(context.Background(), nil)
	if err != nil {
		fmt.Printf("%sError:%s Failed to retrieve cluster IDs: %v\n", colorRed, colorReset, err)
		os.Exit(1)
	}

	// Initialize the KUBECONFIG variable with the existing kubeconfig file
	kubeconfig := kubeconfigPath

	// Create a temporary directory for storing kubeconfig files
	tempDir, err := os.MkdirTemp("", "lke-kubeconfig-")
	if err != nil {
		fmt.Printf("%sError:%s Failed to create temporary directory: %v\n", colorRed, colorReset, err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir)

	// Iterate through each cluster
	for _, cluster := range clusters {
		// Get the cluster details
		clusterDetails, err := linodeClient.GetLKECluster(context.Background(), cluster.ID)
		if err != nil {
			fmt.Printf("%sError:%s Failed to retrieve cluster details for cluster ID %d: %v\n", colorRed, colorReset, cluster.ID, err)
			continue
		}

		// Check if the cluster already exists in the kubeconfig
		cmd := exec.Command("kubectl", "config", "get-contexts", "-o", "name")
		output, err := cmd.Output()
		if err != nil {
			fmt.Printf("%sError:%s Failed to check cluster existence in kubeconfig: %v\n", colorRed, colorReset, err)
			continue
		}
		if strings.Contains(string(output), clusterDetails.Label) {
			fmt.Printf("Cluster %s already exists in the kubeconfig. Skipping...\n", clusterDetails.Label)
			continue
		}

		// Create a temporary kubeconfig file for the cluster
		tempKubeconfig := filepath.Join(tempDir, fmt.Sprintf("kubeconfig-%d", cluster.ID))
		kubeconfigData, err := linodeClient.GetLKEClusterKubeconfig(context.Background(), cluster.ID)
		if err != nil {
			fmt.Printf("%sError:%s Failed to retrieve kubeconfig for cluster ID %d: %v\n", colorRed, colorReset, cluster.ID, err)
			continue
		}
		kubeconfigBytes, err := base64.StdEncoding.DecodeString(kubeconfigData.KubeConfig)
		if err != nil {
			fmt.Printf("%sError:%s Failed to decode kubeconfig for cluster ID %d: %v\n", colorRed, colorReset, cluster.ID, err)
			continue
		}
		err = os.WriteFile(tempKubeconfig, kubeconfigBytes, os.ModePerm)
		if err != nil {
			fmt.Printf("%sError:%s Failed to write temporary kubeconfig file for cluster ID %d: %v\n", colorRed, colorReset, cluster.ID, err)
			continue
		}

		// Update the context and user names in the temporary kubeconfig file
		err = updateKubeconfigNames(tempKubeconfig, clusterDetails.Label)
		if err != nil {
			fmt.Printf("%sError:%s Failed to update names in temporary kubeconfig file for cluster ID %d: %v\n", colorRed, colorReset, cluster.ID, err)
			continue
		}

		kubeconfig = fmt.Sprintf("%s:%s", kubeconfig, tempKubeconfig)
		fmt.Printf("Added cluster %s (Region: %s) to the KUBECONFIG\n", clusterDetails.Label, clusterDetails.Region)
	}

	// Merge the kubeconfig files using kubectl and the KUBECONFIG variable
	cmd := exec.Command("kubectl", "config", "view", "--flatten")
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeconfig))
	mergedKubeconfig, err := cmd.Output()
	if err != nil {
		fmt.Printf("%sError:%s Failed to merge kubeconfig files: %v\n", colorRed, colorReset, err)
		os.Exit(1)
	}
	err = os.WriteFile(kubeconfigPath+".tmp", mergedKubeconfig, os.ModePerm)
	if err != nil {
		fmt.Printf("%sError:%s Failed to write merged kubeconfig file: %v\n", colorRed, colorReset, err)
		os.Exit(1)
	}
	err = os.Rename(kubeconfigPath+".tmp", kubeconfigPath)
	if err != nil {
		fmt.Printf("%sError:%s Failed to replace kubeconfig file: %v\n", colorRed, colorReset, err)
		os.Exit(1)
	}

	fmt.Printf("%sSuccess:%s Kubeconfig updated successfully\n", colorGreen, colorReset)
}

func updateKubeconfigNames(kubeconfigPath, clusterName string) error {
	data, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return err
	}

	var kubeconfig map[string]interface{}
	err = yaml.Unmarshal(data, &kubeconfig)
	if err != nil {
		return err
	}

	contexts, ok := kubeconfig["contexts"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid kubeconfig structure")
	}

	for _, context := range contexts {
		contextMap, ok := context.(map[string]interface{})
		if !ok {
			continue
		}
		contextMap["name"] = clusterName
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
