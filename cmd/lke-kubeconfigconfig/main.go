package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/johnybradshaw/acc-kubeconfig-cli/internal/constants"
	"github.com/johnybradshaw/acc-kubeconfig-cli/internal/kubeconfig"
	"github.com/johnybradshaw/acc-kubeconfig-cli/internal/linode"
	"github.com/johnybradshaw/acc-kubeconfig-cli/pkg/utils"
)

// main is the entry point of the program.
//
// It parses command-line flags, reads Linode tokens from the linode-cli config file or environment variable,
// creates the .kube directory if it doesn't exist, initialises an empty kubeconfig file if it doesn't exist,
// creates Linode clients for each token, retrieves the list of clusters, processes each cluster and merges kubeconfig files,
// and prints a success message.
func main() {
	// Parse command-line flags
	debugModeFlag := flag.Bool("debug", false, "Enable debug mode to print additional information during script execution")
	helpMode := flag.Bool("help", false, "Display help information")
	flag.Parse()

	constants.DebugMode = *debugModeFlag

	if *helpMode {
		utils.DisplayHelp()
		os.Exit(0)
	}

	// Read the Linode tokens from the linode-cli config file
	configTokens, err := readLinodeConfigTokens()
	if err != nil {
		fmt.Printf("%sWarning:%s Failed to read Linode tokens from config file: %v\n", constants.ColorYellow, constants.ColorReset, err)
	}

	// Get the Linode API token from the environment variable
	envToken := os.Getenv("LINODE_API_TOKEN")

	// Use the environment variable token if no tokens found in the config file
	if len(configTokens) == 0 && envToken != "" {
		configTokens = append(configTokens, envToken)
	}

	// Check if any tokens are available
	if len(configTokens) == 0 {
		fmt.Printf("%sError:%s No Linode API tokens found. Please provide a token in the ~/.config/linode-cli file or set the LINODE_API_TOKEN environment variable.\n", constants.ColorRed, constants.ColorReset)
		os.Exit(1)
	}

	// Create the .kube directory if it doesn't exist
	kubeconfig.CreateKubeDirectory()

	// Initialize an empty kubeconfig file if it doesn't exist
	kubeconfigPath := kubeconfig.InitializeKubeconfigFile()

	// Process each token
	for _, token := range configTokens {
		// Create a Linode client
		linodeClient := linode.CreateLinodeClient(token)

		// Get the list of clusters
		clusters, err := linode.GetClusters(linodeClient)
		if err != nil {
			fmt.Printf("%sWarning:%s Failed to retrieve clusters for token: %v\n", constants.ColorYellow, constants.ColorReset, err)
			continue
		}

		// Process each cluster and merge kubeconfig files
		err = linode.ProcessClusterKubeconfigs(linodeClient, clusters, kubeconfigPath, constants.DebugMode)
		if err != nil {
			fmt.Printf("%sWarning:%s Failed to process cluster kubeconfigs for token: %v\n", constants.ColorYellow, constants.ColorReset, err)
		}
	}

	fmt.Printf("%sSuccess:%s Kubeconfig updated successfully\n", constants.ColorGreen, constants.ColorReset)
}

// readLinodeConfigTokens reads the Linode CLI configuration file located at
// $HOME/.config/linode-cli and extracts the API tokens. It returns a slice of
// strings containing the tokens and an error if any occurred.
//
// No parameters.
// Returns a slice of strings and an error.
func readLinodeConfigTokens() ([]string, error) {
	configPath := os.Getenv("HOME") + "/.config/linode-cli"
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var tokens []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "token = ") {
			token := strings.TrimPrefix(line, "token = ")
			tokens = append(tokens, token)
		}
	}

	return tokens, nil
}
