package credentials

import (
	"encoding/json"
	"fmt"
	"kubectm/pkg/utils"
	"os"
	"path/filepath"
	"strings"
)

// azureProfile models the fields kubectm needs from the Azure CLI profile
// file (~/.azure/azureProfile.json) written by `az login`.
type azureProfile struct {
	Subscriptions []azureSubscription `json:"subscriptions"`
}

type azureSubscription struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenantId"`
	IsDefault bool   `json:"isDefault"`
}

// retrieveAzureCredentials retrieves Azure service principal credentials from
// the AZURE_CLIENT_ID / AZURE_CLIENT_SECRET / AZURE_TENANT_ID environment
// variables. The subscription ID comes from AZURE_SUBSCRIPTION_ID or, as a
// fallback, the default subscription in the Azure CLI profile
// (~/.azure/azureProfile.json).
func retrieveAzureCredentials() (*Credential, error) {
	clientID := os.Getenv("AZURE_CLIENT_ID")
	clientSecret := os.Getenv("AZURE_CLIENT_SECRET")
	tenantID := os.Getenv("AZURE_TENANT_ID")

	if clientID == "" || clientSecret == "" || tenantID == "" {
		utils.WarnLogger.Printf("%s No Azure service principal found (set AZURE_CLIENT_ID, AZURE_CLIENT_SECRET and AZURE_TENANT_ID)", utils.Iso8601Time())
		return nil, nil
	}

	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	source := "environment variables"
	if subscriptionID == "" {
		var err error
		subscriptionID, err = azureCLIDefaultSubscription()
		if err != nil {
			return nil, err
		}
		source = "environment variables (subscription from Azure CLI profile)"
	}

	if subscriptionID == "" {
		utils.WarnLogger.Printf("%s Azure service principal found but no subscription is configured (set AZURE_SUBSCRIPTION_ID or run `az login`)", utils.Iso8601Time())
		return nil, nil
	}

	utils.InfoLogger.Printf("%s Azure credentials found via %s", utils.Iso8601Time(), source)

	return &Credential{
		Provider: "Azure",
		Details: map[string]string{
			"ClientID":       clientID,
			"ClientSecret":   clientSecret,
			"TenantID":       tenantID,
			"SubscriptionID": subscriptionID,
		},
	}, nil
}

// azureCLIDefaultSubscription reads the default subscription ID from the
// Azure CLI profile file. Returns an empty string if the file does not exist
// or contains no default subscription.
func azureCLIDefaultSubscription() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("error getting home directory: %v", err)
	}
	homeDir = filepath.Clean(homeDir)

	// Contain the profile path to the home directory. The prefix only gains
	// a separator when it lacks one, so a home directory of "/" still works.
	homePrefix := homeDir
	if !strings.HasSuffix(homePrefix, string(filepath.Separator)) {
		homePrefix += string(filepath.Separator)
	}
	profilePath := filepath.Join(homeDir, ".azure", "azureProfile.json")
	if !strings.HasPrefix(profilePath, homePrefix) {
		return "", fmt.Errorf("invalid Azure profile path")
	}

	data, err := os.ReadFile(profilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("error reading Azure CLI profile: %v", err)
	}

	// The Azure CLI writes azureProfile.json with a UTF-8 byte order mark,
	// which encoding/json rejects, so strip it before parsing.
	data = []byte(strings.TrimPrefix(string(data), "\ufeff"))

	var profile azureProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return "", fmt.Errorf("error parsing Azure CLI profile: %v", err)
	}

	for _, sub := range profile.Subscriptions {
		if sub.IsDefault {
			return sub.ID, nil
		}
	}

	return "", nil
}
