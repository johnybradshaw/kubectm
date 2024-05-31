package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	// Parse command-line flags
	debugMode := flag.Bool("debug", false, "Enable debug mode to print additional information during script execution")
	helpMode := flag.Bool("help", false, "Display help information")
	flag.Parse()

	if *helpMode {
		displayHelp()
		os.Exit(0)
	}

	// Get the Linode API token from the environment variable
	apiToken := os.Getenv("LINODE_API_TOKEN")
	if apiToken == "" {
		fmt.Printf("%sError:%s %sLINODE_API_TOKEN%s environment variable is not set. Please set it and run the script again.\n", colorRed, colorReset, colorCyan, colorReset)
		os.Exit(1)
	}

	// Check if required dependencies are installed
	checkDependencies()

	// Create the .kube directory if it doesn't exist
	createKubeDirectory()

	// Initialize an empty kubeconfig file if it doesn't exist
	kubeconfigPath := initializeKubeconfigFile()

	// Create a Linode client
	linodeClient := createLinodeClient(apiToken)

	// Get the list of clusters
	clusters, err := getClusters(linodeClient)
	if err != nil {
		fmt.Printf("%sError:%s Failed to retrieve clusters: %v\n", colorRed, colorReset, err)
		os.Exit(1)
	}

	// Process each cluster and merge kubeconfig files
	err = processClusterKubeconfigs(linodeClient, clusters, kubeconfigPath, *debugMode)
	if err != nil {
		fmt.Printf("%sError:%s Failed to process cluster kubeconfigs: %v\n", colorRed, colorReset, err)
		os.Exit(1)
	}

	fmt.Printf("%sSuccess:%s Kubeconfig updated successfully\n", colorGreen, colorReset)
}
