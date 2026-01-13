package main

import (
    "flag"
    "kubectm/pkg/credentials"
    "kubectm/pkg/kubeconfig"
    "kubectm/pkg/ui"
    "log"
    "os"
    "path/filepath"
    "encoding/json"
    "time"
    "github.com/fatih/color"
)

const (
    // Version is set during build time using -ldflags
    version         = "development"
    storedCredsPath = ".kubectm/selected_credentials.json"
)

var (
    infoLogger    = log.New(os.Stdout, color.GreenString("[INFO] "), 0)
    warnLogger    = log.New(os.Stdout, color.YellowString("[WARN] "), 0)
    errorLogger   = log.New(os.Stderr, color.RedString("[ERROR] "), 0)
    actionLogger  = log.New(os.Stdout, color.CyanString("[ACTION] "), 0)
)

// iso8601Time returns the current time in the ISO 8601 format
func iso8601Time() string {
    // Format the current time according to the ISO 8601 standard
    // Example: 2022-11-19T15:03:52Z
    return time.Now().Format(time.RFC3339)
}

// printUsage prints the usage message for the kubectm command.
func printUsage() {
    // Print the usage message
    color.Cyan(`kubectm - A tool to download and integrate Kubernetes configurations across multiple cloud providers.

Usage: kubectm [options]

Options:
  -h, --help        Show this help message and exit.
  -v, --version     Show the version of kubectm.
  --reset-creds     Reset the stored credentials and prompt for new ones.

For more information and source code, visit:
https://github.com/johnybradshaw/kubectm
`)
}

// SaveSelectedCredentialProviders saves the selected credential providers to the specified file.
//
// It takes a slice of provider names as an argument.
//
// The function creates the specified file if it doesn't exist, and overwrites
// the file if it does exist.
//
// The function returns an error if there is a problem writing the file.
func SaveSelectedCredentialProviders(providers []string) error {
    // Get the user's home directory
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return err
    }

    // Create the .kubectm directory if it doesn't exist
    configDir := filepath.Join(homeDir, ".kubectm")
    err = os.MkdirAll(configDir, os.ModePerm)
    if err != nil {
        return err
    }

    // Create the selected_providers.json file if it doesn't exist, or overwrite it if it does
    configFile := filepath.Join(configDir, "selected_providers.json")
    file, err := os.Create(configFile)
    if err != nil {
        return err
    }
    defer file.Close()

    // Encode the selected providers as JSON and write them to the file
    encoder := json.NewEncoder(file)
    return encoder.Encode(providers)
}

// LoadSelectedCredentialProviders loads the selected credential providers from the
// ~/.kubectm/selected_providers.json file.
//
// It returns a slice of provider names and an error if the file
// doesn't exist or there is a problem decoding the JSON.
func LoadSelectedCredentialProviders() ([]string, error) {
    // Get the user's home directory
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return nil, err
    }

    // Construct the path to the selected_providers.json file
    configFile := filepath.Join(homeDir, ".kubectm", "selected_providers.json")

    // Open the file for reading
    file, err := os.Open(configFile)
    if err != nil {
        // If the file doesn't exist or there is a problem reading the file,
        // return an error
        return nil, err
    }
    defer file.Close()

    // Decode the JSON in the file into a slice of provider names
    var providers []string
    decoder := json.NewDecoder(file)
    err = decoder.Decode(&providers)
    if err != nil {
        // If there is a problem decoding the JSON, return an error
        return nil, err
    }

    // Return the slice of provider names
    return providers, nil
}

// resetStoredCredentials removes the stored credentials file to force re-prompting
func resetStoredCredentials() {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        errorLogger.Fatalf("%s Failed to get user home directory: %v", iso8601Time(), err)
    }
    configFile := filepath.Join(homeDir, storedCredsPath)
    err = os.Remove(configFile)
    if err != nil && !os.IsNotExist(err) {
        errorLogger.Fatalf("%s Failed to reset stored credentials: %v", iso8601Time(), err)
    }
    warnLogger.Printf("%s Stored credentials have been reset. You'll be prompted to select credentials.", iso8601Time())
}

// promptAndSelectProviders prompts the user to select credential providers and saves their selection
func promptAndSelectProviders() []string {
    creds, err := credentials.RetrieveAll()
    if err != nil {
        errorLogger.Fatalf("%s Failed to retrieve credentials: %v", iso8601Time(), err)
    }

    selectedCreds := ui.SelectCredentials(creds)

    providers := make([]string, 0, len(selectedCreds))
    for _, cred := range selectedCreds {
        providers = append(providers, cred.Provider)
    }

    if err := SaveSelectedCredentialProviders(providers); err != nil {
        errorLogger.Printf("%s Failed to save selected providers: %v", iso8601Time(), err)
    }

    return providers
}

// getSelectedProviders loads saved providers or prompts the user to select them
func getSelectedProviders() []string {
    selectedProviders, err := LoadSelectedCredentialProviders()
    if err != nil || len(selectedProviders) == 0 {
        warnLogger.Printf("%s No previous credential selections found or an error occurred, prompting user to select credentials.", iso8601Time())
        return promptAndSelectProviders()
    }
    infoLogger.Printf("%s Using previously selected credential providers.", iso8601Time())
    return selectedProviders
}

// downloadAllConfigs downloads kubeconfig files for all provided credentials
func downloadAllConfigs(creds []credentials.Credential) {
    for _, cred := range creds {
        infoLogger.Printf("%s Downloading kubeconfig from %s", iso8601Time(), cred.Provider)
        if err := kubeconfig.DownloadConfigs([]credentials.Credential{cred}); err != nil {
            errorLogger.Fatalf("%s Failed to download kubeconfig files from %s: %v", iso8601Time(), cred.Provider, err)
        }
    }
}

func main() {
    var showHelp bool
    var showVersion bool
    var resetCreds bool

    flag.BoolVar(&showHelp, "help", false, "Show help message")
    flag.BoolVar(&showHelp, "h", false, "Show help message")
    flag.BoolVar(&showVersion, "version", false, "Show version information")
    flag.BoolVar(&showVersion, "v", false, "Show version information")
    flag.BoolVar(&resetCreds, "reset-creds", false, "Reset stored credentials and prompt for new ones")
    flag.Parse()

    if showHelp {
        printUsage()
        os.Exit(0)
    }

    if showVersion {
        color.Cyan("kubectm version %s\n", version)
        os.Exit(0)
    }

    infoLogger.Printf("%s Starting kubectm...\n", iso8601Time())

    if resetCreds {
        resetStoredCredentials()
    }

    selectedProviders := getSelectedProviders()

    creds, err := credentials.RetrieveSelected(selectedProviders)
    if err != nil {
        errorLogger.Fatalf("%s Failed to retrieve selected credentials: %v", iso8601Time(), err)
    }

    downloadAllConfigs(creds)

    if err := kubeconfig.MergeConfigs(); err != nil {
        errorLogger.Fatalf("%s Failed to merge kubeconfig files: %v", iso8601Time(), err)
    }

    infoLogger.Printf("%s kubectm finished successfully.", iso8601Time())
}
