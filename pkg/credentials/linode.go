package credentials

import (
    "bufio"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "strings"
    "io/ioutil"
)

func retrieveLinodeCredentials() (*Credential, error) {
    // First, try to retrieve the access token from environment variables
    accessToken := os.Getenv("LINODE_ACCESS_TOKEN")
    if accessToken != "" {
        return &Credential{
            Provider: "Linode",
            Details: map[string]string{
                "AccessToken": accessToken,
            },
        }, nil
    }

    // Correct path: ~/.config/linode-cli
    configDirPath := filepath.Join(os.Getenv("HOME"), ".config", "linode-cli")
    log.Printf("Looking for Linode config in directory: %s", configDirPath)

    // Check if the directory exists
    _, err := os.Stat(configDirPath)
    if os.IsNotExist(err) {
        log.Printf("Linode config directory not found: %v", err)
        return nil, fmt.Errorf("linode config directory not found")
    }

    // Load and parse the credentials from the directory content
    configFileContent, err := ioutil.ReadFile(configDirPath)
    if err != nil {
        log.Printf("Error reading Linode config directory: %v", err)
        return nil, fmt.Errorf("error reading Linode config")
    }

    // Find the default profile from the config file content
    defaultProfile := getDefaultProfile(configFileContent)
    log.Printf("Default profile found: %s", defaultProfile)
    if defaultProfile == "" {
        return nil, fmt.Errorf("no default profile found in Linode config")
    }

    // Parse the config file content to extract the access token from the default profile
    accessToken = parseLinodeConfig(configFileContent, defaultProfile)
    if accessToken != "" {
        log.Printf("Access token found: %s", accessToken)
        return &Credential{
            Provider: "Linode",
            Details: map[string]string{
                "AccessToken": accessToken,
            },
        }, nil
    }

    // If credentials are still not found, return an error
    return nil, fmt.Errorf("linode credentials not found")
}

// getDefaultProfile extracts the default profile name from the [DEFAULT] section
func getDefaultProfile(configContent []byte) string {
    scanner := bufio.NewScanner(strings.NewReader(string(configContent)))
    inDefaultSection := false
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line == "[DEFAULT]" {
            inDefaultSection = true
            continue
        }

        // Break out of the loop if another section starts
        if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
            inDefaultSection = false
        }

        if inDefaultSection && strings.HasPrefix(line, "default-user") {
            parts := strings.Split(line, "=")
            if len(parts) == 2 {
                return strings.TrimSpace(parts[1])
            }
        }
    }
    return ""
}

// parseLinodeConfig extracts the access token from the specified profile section
func parseLinodeConfig(configContent []byte, profile string) string {
    scanner := bufio.NewScanner(strings.NewReader(string(configContent)))
    inSection := false
    sectionHeader := fmt.Sprintf("[%s]", profile)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        log.Printf("Parsing line: %s", line)
        if line == sectionHeader {
            log.Printf("Entering section: %s", profile)
            inSection = true
            continue
        } else if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
            log.Printf("Exiting section: %s", profile)
            inSection = false
        }

        if inSection && strings.HasPrefix(line, "token") {
            parts := strings.Split(line, "=")
            if len(parts) == 2 {
                return strings.TrimSpace(parts[1])
            }
        }
    }
    return ""
}