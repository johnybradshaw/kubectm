package credentials

import (
	"bufio"
	"encoding/json"
	"fmt"
	"kubectm/pkg/utils"
	"os"
	"path/filepath"
	"strings"
)

// gcpCredentialsJSON models the fields kubectm needs from a Google Cloud
// credentials JSON file (service account key or gcloud application default
// credentials).
type gcpCredentialsJSON struct {
	Type           string `json:"type"`
	ProjectID      string `json:"project_id"`
	ClientEmail    string `json:"client_email"`
	QuotaProjectID string `json:"quota_project_id"`
}

// retrieveGCPCredentials retrieves GCP credentials from the
// GOOGLE_APPLICATION_CREDENTIALS environment variable or the gcloud CLI
// application default credentials file. The project ID is resolved from the
// GOOGLE_CLOUD_PROJECT env var, the credentials JSON, or the gcloud CLI
// active configuration, in that order.
func retrieveGCPCredentials() (*Credential, error) {
	credsPath, source, err := findGCPCredentialsFile()
	if err != nil {
		return nil, err
	}
	if credsPath == "" {
		utils.WarnLogger.Printf("%s No GCP credentials found (set GOOGLE_APPLICATION_CREDENTIALS or run `gcloud auth application-default login`)", utils.Iso8601Time())
		return nil, nil
	}

	data, err := os.ReadFile(credsPath)
	if err != nil {
		return nil, fmt.Errorf("error reading GCP credentials file %s: %v", credsPath, err)
	}

	var credsJSON gcpCredentialsJSON
	if err := json.Unmarshal(data, &credsJSON); err != nil {
		return nil, fmt.Errorf("error parsing GCP credentials file %s: %v", credsPath, err)
	}

	projectID := resolveGCPProjectID(credsJSON)
	if projectID == "" {
		utils.WarnLogger.Printf("%s GCP credentials found via %s but no project ID is configured (set GOOGLE_CLOUD_PROJECT or run `gcloud config set project`)", utils.Iso8601Time(), source)
		return nil, nil
	}

	utils.InfoLogger.Printf("%s GCP credentials found via %s", utils.Iso8601Time(), source)

	details := map[string]string{
		"CredentialsFile": credsPath,
		"ProjectID":       projectID,
	}
	if credsJSON.ClientEmail != "" {
		details["ClientEmail"] = credsJSON.ClientEmail
	}

	return &Credential{
		Provider: "GCP",
		Details:  details,
	}, nil
}

// findGCPCredentialsFile locates a Google credentials JSON file, checking the
// GOOGLE_APPLICATION_CREDENTIALS environment variable first and then the
// gcloud CLI application default credentials path. Returns an empty path if
// no credentials file is found.
func findGCPCredentialsFile() (path, source string, err error) {
	if envPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); envPath != "" {
		if _, statErr := os.Stat(envPath); statErr != nil {
			return "", "", fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS points to an unreadable file: %v", statErr)
		}
		return envPath, "GOOGLE_APPLICATION_CREDENTIALS environment variable", nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("error getting home directory: %v", err)
	}
	homeDir = filepath.Clean(homeDir)

	adcPath := filepath.Clean(filepath.Join(homeDir, ".config", "gcloud", "application_default_credentials.json"))
	if !strings.HasPrefix(adcPath, homeDir+string(filepath.Separator)) {
		return "", "", fmt.Errorf("invalid credentials file path")
	}
	if _, statErr := os.Stat(adcPath); statErr != nil {
		if os.IsNotExist(statErr) {
			return "", "", nil
		}
		return "", "", fmt.Errorf("error reading gcloud application default credentials: %v", statErr)
	}
	return adcPath, "gcloud application default credentials", nil
}

// resolveGCPProjectID determines the GCP project to scan for clusters:
// GOOGLE_CLOUD_PROJECT env var, then the credentials JSON itself, then the
// gcloud CLI active configuration.
func resolveGCPProjectID(credsJSON gcpCredentialsJSON) string {
	if project := os.Getenv("GOOGLE_CLOUD_PROJECT"); project != "" {
		return project
	}
	if credsJSON.ProjectID != "" {
		return credsJSON.ProjectID
	}
	if credsJSON.QuotaProjectID != "" {
		return credsJSON.QuotaProjectID
	}
	return gcloudActiveProject()
}

// gcloudActiveProject reads the active project from the gcloud CLI config
// (~/.config/gcloud/configurations/config_default, [core] section, project
// key). Returns an empty string if not configured.
func gcloudActiveProject() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	homeDir = filepath.Clean(homeDir)

	configPath := filepath.Clean(filepath.Join(homeDir, ".config", "gcloud", "configurations", "config_default"))
	if !strings.HasPrefix(configPath, homeDir+string(filepath.Separator)) {
		return ""
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}

	return parseGcloudProject(content)
}

// parseGcloudProject extracts the project value from the [core] section of a
// gcloud INI-style configuration file.
func parseGcloudProject(content []byte) string {
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	inCore := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			inCore = line == "[core]"
			continue
		}

		if inCore {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			if strings.TrimSpace(parts[0]) == "project" {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	return ""
}
