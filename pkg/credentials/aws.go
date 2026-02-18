package credentials

import (
	"bufio"
	"fmt"
	"kubectm/pkg/utils"
	"os"
	"path/filepath"
	"strings"
)

// retrieveAWSCredentials retrieves AWS credentials from environment variables
// or the ~/.aws/credentials file.
func retrieveAWSCredentials() (*Credential, error) {
	// Check environment variables first
	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	if accessKeyID != "" && secretAccessKey != "" {
		details := map[string]string{
			"AccessKey": accessKeyID,
			"SecretKey": secretAccessKey,
		}

		if sessionToken := os.Getenv("AWS_SESSION_TOKEN"); sessionToken != "" {
			details["SessionToken"] = sessionToken
		}
		if region := os.Getenv("AWS_DEFAULT_REGION"); region != "" {
			details["Region"] = region
		}

		utils.InfoLogger.Printf("%s AWS credentials found via environment variables", utils.Iso8601Time())

		return &Credential{
			Provider: "AWS",
			Details:  details,
		}, nil
	}

	// Fall back to ~/.aws/credentials file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("error getting home directory: %v", err)
	}
	credentialsFilePath := filepath.Join(homeDir, ".aws", "credentials")

	// Validate that the resolved path is within the home directory to prevent path traversal
	if !strings.HasPrefix(credentialsFilePath, homeDir+string(filepath.Separator)) {
		return nil, fmt.Errorf("invalid credentials file path")
	}

	utils.InfoLogger.Printf("%s Looking for AWS credentials file", utils.Iso8601Time())

	fileContent, err := os.ReadFile(credentialsFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			utils.WarnLogger.Printf("%s AWS credentials file not found", utils.Iso8601Time())
			return nil, nil
		}
		utils.ErrorLogger.Printf("%s Error reading AWS credentials file: %v", utils.Iso8601Time(), err)
		return nil, fmt.Errorf("error reading AWS credentials file: %v", err)
	}

	profile := os.Getenv("AWS_PROFILE")
	if profile == "" {
		profile = "default"
	}

	cred := parseAWSCredentialsFile(fileContent, profile)
	if cred != nil {
		return cred, nil
	}

	utils.WarnLogger.Printf("%s No AWS credentials found for profile %q", utils.Iso8601Time(), profile)
	return nil, nil
}

// parseAWSCredentialsFile parses the AWS credentials file content for the given profile.
// The file follows INI format with sections like [default] or [profile-name].
func parseAWSCredentialsFile(content []byte, profile string) *Credential {
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	inSection := false
	sectionHeader := fmt.Sprintf("[%s]", profile)

	var accessKeyID, secretAccessKey, sessionToken, region string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if line == sectionHeader {
			utils.InfoLogger.Printf("%s Entering AWS credentials section: %s", utils.Iso8601Time(), profile)
			inSection = true
			continue
		} else if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			if inSection {
				// We've reached the next section, stop parsing
				break
			}
			continue
		}

		if inSection {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch strings.ToLower(key) {
			case "aws_access_key_id":
				accessKeyID = value
			case "aws_secret_access_key":
				secretAccessKey = value
			case "aws_session_token":
				sessionToken = value
			case "region":
				region = value
			}
		}
	}

	if accessKeyID == "" || secretAccessKey == "" {
		return nil
	}

	details := map[string]string{
		"AccessKey": accessKeyID,
		"SecretKey": secretAccessKey,
	}
	if sessionToken != "" {
		details["SessionToken"] = sessionToken
	}
	if region != "" {
		details["Region"] = region
	}

	utils.InfoLogger.Printf("%s AWS credentials found via credentials file", utils.Iso8601Time())

	return &Credential{
		Provider: "AWS",
		Details:  details,
	}
}
