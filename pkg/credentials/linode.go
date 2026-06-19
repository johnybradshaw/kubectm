package credentials

import (
    "bufio"
    "fmt"
    "kubectm/pkg/utils" // Import your utils package for logging
    "os"
    "path/filepath"
    "strings"
)

// retrieveLinodeCredentials retrieves Linode credentials
func retrieveLinodeCredentials() (*Credential, error) {
    accessToken := os.Getenv("LINODE_ACCESS_TOKEN")
    if accessToken != "" {
        obfuscatedToken := utils.ObfuscateCredential(accessToken)
        utils.InfoLogger.Printf("%s Linode credentials found: %v", utils.Iso8601Time(), map[string]string{
            "AccessToken": obfuscatedToken,
        })
        return &Credential{
            Provider: "Linode",
            Details: map[string]string{
                "AccessToken": accessToken,
            },
        }, nil
    }

    homeDir, err := os.UserHomeDir()
    if err != nil {
        utils.ErrorLogger.Printf("%s Error determining home directory: %v", utils.Iso8601Time(), err)
        return nil, fmt.Errorf("error determining home directory")
    }
    absHomeDir, err := filepath.Abs(filepath.Clean(homeDir))
    if err != nil {
        utils.ErrorLogger.Printf("%s Error resolving home directory: %v", utils.Iso8601Time(), err)
        return nil, fmt.Errorf("error resolving home directory")
    }
    configDirPath := filepath.Clean(filepath.Join(absHomeDir, ".config", "linode-cli"))
    // Confirm the resolved path stays within the home directory before any
    // filesystem access (defends against a tampered HOME environment variable).
    if rel, relErr := filepath.Rel(absHomeDir, configDirPath); relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
        return nil, fmt.Errorf("invalid linode config path: %s", configDirPath)
    }
    utils.InfoLogger.Printf("%s Looking for Linode config in directory: %s", utils.Iso8601Time(), configDirPath)

    _, err = os.Stat(configDirPath)
    if os.IsNotExist(err) {
        utils.WarnLogger.Printf("%s Linode config directory not found: %v", utils.Iso8601Time(), err)
        return nil, fmt.Errorf("linode config directory not found")
    }

    configFileContent, err := os.ReadFile(configDirPath)
    if err != nil {
        utils.ErrorLogger.Printf("%s Error reading Linode config directory: %v", utils.Iso8601Time(), err)
        return nil, fmt.Errorf("error reading Linode config")
    }

    defaultProfile := getDefaultProfile(configFileContent)
    utils.InfoLogger.Printf("%s Default profile found: %s", utils.Iso8601Time(), defaultProfile)
    if defaultProfile == "" {
        return nil, fmt.Errorf("no default profile found in Linode config")
    }

    accessToken = parseLinodeConfig(configFileContent, defaultProfile)
    if accessToken != "" {
        obfuscatedToken := utils.ObfuscateCredential(accessToken)
        //utils.InfoLogger.Printf("%s Access token found: %s", utils.Iso8601Time(), obfuscatedToken)
        utils.InfoLogger.Printf("%s Linode credentials found: %v", utils.Iso8601Time(), map[string]string{
            "AccessToken": obfuscatedToken,
        })
        return &Credential{
            Provider: "Linode",
            Details: map[string]string{
                "AccessToken": accessToken,
            },
        }, nil
    }

    return nil, fmt.Errorf("linode credentials not found")
}

// parseLinodeConfig extracts the access token from the specified profile section
func parseLinodeConfig(configContent []byte, profile string) string {
    scanner := bufio.NewScanner(strings.NewReader(string(configContent)))
    inSection := false
    sectionHeader := fmt.Sprintf("[%s]", profile)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line == sectionHeader {
            utils.InfoLogger.Printf("%s Entering section: %s", utils.Iso8601Time(), profile)
            inSection = true
            continue
        } else if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
            utils.InfoLogger.Printf("%s Exiting section: %s", utils.Iso8601Time(), profile)
            inSection = false
        }

        // Match the "token" key exactly (case-insensitive) and split into at
        // most two parts so an "=" inside the value is preserved. Other config
        // lines may contain sensitive values, so they are never logged.
        if inSection {
            parts := strings.SplitN(line, "=", 2)
            if len(parts) == 2 && strings.TrimSpace(strings.ToLower(parts[0])) == "token" {
                token := strings.TrimSpace(parts[1])
                obfuscatedToken := utils.ObfuscateCredential(token)
                utils.InfoLogger.Printf("%s Access token found: %s", utils.Iso8601Time(), obfuscatedToken)
                return token
            }
        }
    }
    return ""
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
