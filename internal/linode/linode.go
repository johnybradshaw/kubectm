package linode

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

	// Import local packages
	"github.com/johnybradshaw/acc-kubeconfig-cli/internal/constants"
	kc "github.com/johnybradshaw/acc-kubeconfig-cli/internal/kubeconfig"
	"github.com/johnybradshaw/acc-kubeconfig-cli/pkg/utils"
)

// CreateLinodeClient creates a Linode client using the provided API token.
//
// Parameters:
// - apiToken: The API token used to authenticate with the Linode API.
//
// Returns:
// - linodego.Client: The Linode client instance.
func CreateLinodeClient(apiToken string) linodego.Client {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: apiToken})
	oauth2Client := &http.Client{
		Transport: &oauth2.Transport{
			Source: tokenSource,
		},
	}
	return linodego.NewClient(oauth2Client)
}

// GetClusters retrieves a list of LKECluster objects from the Linode API.
//
// Parameters:
// - linodeClient: The Linode client instance used to make the API request.
//
// Returns:
// - []linodego.LKECluster: A slice of LKECluster objects representing the clusters retrieved from the API.
// - error: An error if the API request fails.
func GetClusters(linodeClient linodego.Client) ([]linodego.LKECluster, error) {
	clusters, err := linodeClient.ListLKEClusters(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	return clusters, nil
}

// ProcessClusterKubeconfigs processes the kubeconfigs for each cluster and merges them into a single kubeconfig file.
//
// Parameters:
// - linodeClient: The Linode client instance used to retrieve cluster details and kubeconfigs.
// - clusters: The list of LKECluster objects representing the clusters to process.
// - kubeconfigPath: The path to the existing kubeconfig file.
// - debugMode: A boolean indicating whether debug mode is enabled.
//
// Returns:
// - error: An error if any of the API requests or file operations fail.
func ProcessClusterKubeconfigs(linodeClient linodego.Client, clusters []linodego.LKECluster, kubeconfigPath string, debugMode bool) error {
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
			fmt.Printf("%sError:%s Failed to retrieve cluster details for cluster ID %d: %v\n", constants.ColorRed, constants.ColorReset, cluster.ID, err)
			continue
		}
		utils.PrintDebug("Processing cluster: %s (ID: %d)", clusterDetails.Label, cluster.ID, debugMode)

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
			fmt.Printf("%sError:%s Failed to retrieve kubeconfig for cluster ID %d: %v\n", constants.ColorRed, constants.ColorReset, cluster.ID, err)
			continue
		}
		kubeconfigBytes, err := base64.StdEncoding.DecodeString(kubeconfigData.KubeConfig)
		if err != nil {
			fmt.Printf("%sError:%s Failed to decode kubeconfig for cluster ID %d: %v\n", constants.ColorRed, constants.ColorReset, cluster.ID, err)
			continue
		}
		err = os.WriteFile(tempKubeconfig, kubeconfigBytes, os.ModePerm)
		if err != nil {
			fmt.Printf("%sError:%s Failed to write temporary kubeconfig file for cluster ID %d: %v\n", constants.ColorRed, constants.ColorReset, cluster.ID, err)
			continue
		}
		utils.PrintDebug("Temporary kubeconfig file created: %s", tempKubeconfig, debugMode)

		// Update the context name in the temporary kubeconfig file
		err = kc.UpdateContextName(tempKubeconfig, oldContextName, clusterDetails.Label)
		if err != nil {
			fmt.Printf("%sError:%s Failed to update context name in temporary kubeconfig file for cluster ID %d: %v\n", constants.ColorRed, constants.ColorReset, cluster.ID, err)
			continue
		}

		kubeconfig = fmt.Sprintf("%s:%s", kubeconfig, tempKubeconfig)
		fmt.Printf("Added cluster %s (Region: %s) to the KUBECONFIG\n", clusterDetails.Label, clusterDetails.Region)
	}

	// Merge the kubeconfig files using kubectl and the KUBECONFIG variable
	err = kc.MergeKubeconfigs(kubeconfig, kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to merge kubeconfig files: %v", err)
	}

	return nil
}

// isContextExists checks if a context with the given name exists in the kubeconfig file.
//
// Parameters:
// - kubeconfigPath: The path to the kubeconfig file.
// - contextName: The name of the context to check.
//
// Returns:
// - bool: True if the context exists, false otherwise.
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
