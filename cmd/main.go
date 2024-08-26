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
    version         = "1.0.0"
    storedCredsPath = ".kubectm/selected_credentials.json"
)

var (
    infoLogger    = log.New(os.Stdout, color.GreenString("[INFO] "), 0)
    warnLogger    = log.New(os.Stdout, color.YellowString("[WARN] "), 0)
    errorLogger   = log.New(os.Stderr, color.RedString("[ERROR] "), 0)
    actionLogger  = log.New(os.Stdout, color.CyanString("[ACTION] "), 0)
)

func iso8601Time() string {
    return time.Now().Format(time.RFC3339)
}

func printUsage() {
    color.Cyan(`kubectm - A tool to manage Kubernetes configurations across multiple cloud providers.

Usage: kubectm [options]

Options:
  -h, --help        Show this help message and exit.
  -v, --version     Show the version of kubectm.
  --reset-creds     Reset the stored credentials and prompt for new ones.
`)
}

func SaveSelectedCredentials(creds []credentials.Credential) error {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return err
    }

    configDir := filepath.Join(homeDir, ".kubectm")
    configFile := filepath.Join(configDir, "selected_credentials.json")

    err = os.MkdirAll(configDir, os.ModePerm)
    if err != nil {
        return err
    }

    file, err := os.Create(configFile)
    if err != nil {
        return err
    }
    defer file.Close()

    encoder := json.NewEncoder(file)
    return encoder.Encode(creds)
}

func LoadSelectedCredentials() ([]credentials.Credential, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return nil, err
    }

    configFile := filepath.Join(homeDir, ".kubectm", "selected_credentials.json")

    file, err := os.Open(configFile)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    var creds []credentials.Credential
    decoder := json.NewDecoder(file)
    err = decoder.Decode(&creds)
    if err != nil {
        return nil, err
    }

    return creds, nil
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

    // Try to load previously selected credentials
    selectedCreds, err := LoadSelectedCredentials()
    if err != nil || len(selectedCreds) == 0 {
        warnLogger.Printf("%s No previous credentials found or an error occurred, prompting user to select credentials.", iso8601Time())
        
        creds, err := credentials.RetrieveAll()
        if err != nil {
            errorLogger.Fatalf("%s Failed to retrieve credentials: %v", iso8601Time(), err)
        }

        selectedCreds = ui.SelectCredentials(creds)

        // Save the selected credentials for future use
        err = SaveSelectedCredentials(selectedCreds)
        if err != nil {
            errorLogger.Printf("%s Failed to save selected credentials: %v", iso8601Time(), err)
        }
    } else {
        infoLogger.Printf("%s Using previously selected credentials.", iso8601Time())
    }

    for _, cred := range selectedCreds {
        infoLogger.Printf("%s Downloading kubeconfig from %s", iso8601Time(), cred.Provider)
        err := kubeconfig.DownloadConfigs([]credentials.Credential{cred})
        if err != nil {
            errorLogger.Fatalf("%s Failed to download kubeconfig files from %s: %v", iso8601Time(), cred.Provider, err)
        }
    }

    err = kubeconfig.MergeConfigs()
    if err != nil {
        errorLogger.Fatalf("%s Failed to merge kubeconfig files: %v", iso8601Time(), err)
    }

    infoLogger.Printf("%s kubectm finished successfully.", iso8601Time())
}