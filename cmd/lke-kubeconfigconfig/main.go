package main

import (
	"flag"
	"fmt"
	"os"

	// Import local packages
    "github.com/johnybradshaw/acc-kubeconfig-cli/internal/constants"
    kc "github.com/johnybradshaw/acc-kubeconfig-cli/internal/kubeconfig"
    "github.com/johnybradshaw/acc-kubeconfig-cli/internal/linode"
    "github.com/johnybradshaw/acc-kubeconfig-cli/pkg/utils"
)
func main() {
	// Parse command-line flags
	debugModeFlag := flag.Bool("debug", false, "Enable debug mode to print additional information during script execution")
	helpMode := flag.Bool("help", false, "Display help information")
	flag.Parse()

	constants.DebugMode = *debugModeFlag

	// Display help information if the --help flag is set
	if *helpMode {
		utils.DisplayHelp()
		os.Exit(0)
	}

	// Get the Linode API token from the environment variable
	apiToken := os.Getenv("LINODE_API_TOKEN")
	if apiToken == "" {
		fmt.Printf("%sError:%s %sLINODE_API_TOKEN%s environment variable is not set. Please set it and run the script again.\n", constants.ColorRed, constants.ColorReset, constants.ColorCyan, constants.ColorReset)
		os.Exit(1)
	}

	// Check if required dependencies are installed
	utils.CheckDependencies()

	// Create the .kube directory if it doesn't exist
	kc.CreateKubeDirectory()

	// Initialize an empty kubeconfig file if it doesn't exist
	kubeconfigPath := kc.InitializeKubeconfigFile()

	// Create a Linode client
	linodeClient := linode.CreateLinodeClient(apiToken)

	// Get the list of clusters
	clusters, err := linode.GetClusters(linodeClient)
	if err != nil {
		fmt.Printf("%sError:%s Failed to retrieve clusters: %v\n", constants.ColorRed, constants.ColorReset, err)
		os.Exit(1)
	}

	// Process each cluster and merge kubeconfig files
	err = linode.ProcessClusterKubeconfigs(linodeClient, clusters, kubeconfigPath, constants.DebugMode)
	if err != nil {
		fmt.Printf("%sError:%s Failed to process cluster kubeconfigs: %v\n", constants.ColorRed, constants.ColorReset, err)
		os.Exit(1)
	}

	fmt.Printf("%sSuccess:%s Kubeconfig updated successfully\n", constants.ColorGreen, constants.ColorReset)
}
