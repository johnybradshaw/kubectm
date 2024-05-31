package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/linode/linodego"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
)

func createLinodeClient(apiToken string) linodego.Client {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: apiToken})
	oauth2Client := &http.Client{
		Transport: &oauth2.Transport{
			Source: tokenSource,
		},
	}
	return linodego.NewClient(oauth2Client)
}

func getClusters(linodeClient linodego.Client) ([]linodego.LKECluster, error) {
	clusters, err := linodeClient.ListLKEClusters(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	return clusters, nil
}

func processClusterKubeconfigs(linodeClient linodego.Client, clusters []linodego.LKECluster, kubeconfigPath string, debugMode bool) error {
	// Initialize the KUBECONFIG variable with the existing kubeconfig file
	kubeconfig := kubeconfigPath

	// Create a temporary directory for storing kubeconfig files
	tempDir, err := os.MkdirTemp("", "lke-kubeconfig-")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %v", err)
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
		printDebug("Processing cluster: %s (ID: %d)", clusterDetails.Label, cluster.ID, debugMode)

		// Check if the cluster already exists in the kubeconfig
		oldContextName := fmt.Sprintf("lke%d-ctx", cluster.ID)
		if isContextExists(kubeconfigPath, oldContextName) {
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
		printDebug("Temporary kubeconfig file created: %s", tempKubeconfig, debugMode)

		// Update the context name in the temporary kubeconfig file
		err = updateContextName(tempKubeconfig, oldContextName, clusterDetails.Label)
		if err != nil {
			fmt.Printf("%sError:%s Failed to update context name in temporary kubeconfig file for cluster ID %d: %v\n", colorRed, colorReset, cluster.ID, err)
			continue
		}

		kubeconfig = fmt.Sprintf("%s:%s", kubeconfig, tempKubeconfig)
		fmt.Printf("Added cluster %s (Region: %s) to the KUBECONFIG\n", clusterDetails.Label, clusterDetails.Region)
	}

	// Merge the kubeconfig files using kubectl and the KUBECONFIG variable
	err = mergeKubeconfigs(kubeconfig, kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to merge kubeconfig files: %v", err)
	}

	return nil
}

func isContextExists(kubeconfigPath, contextName string) bool {
	data, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return false
	}

	var kubeconfig map[string]interface{}
	err = yaml.Unmarshal(data, &kubeconfig)
	if err != nil {
		return false
	}

	// Check if the context exists
	contexts, ok := kubeconfig["contexts"].([]interface{})
	if ok {
		for _, context := range contexts {
			contextMap, ok := context.(map[string]interface{})
			if ok {
				if contextMap["name"] == contextName {
					return true
				}
			}
		}
	}

	return false
}
