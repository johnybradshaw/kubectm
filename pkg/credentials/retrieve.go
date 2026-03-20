package credentials

import (
	"errors"
	"fmt"
	"kubectm/pkg/utils"
)

type Credential struct {
	Provider string
	Details  map[string]string
}

// logCredentialDiscovery logs the discovery of credentials for a provider,
// obfuscating all credential values before logging.
func logCredentialDiscovery(provider string, cred *Credential) {
	if cred == nil {
		return
	}
	obfuscated := make(map[string]string, len(cred.Details))
	for k, v := range cred.Details {
		obfuscated[k] = utils.ObfuscateCredential(v)
	}
	utils.InfoLogger.Printf("%s %s credentials found: %v", utils.Iso8601Time(), provider, obfuscated)
}

// RetrieveAll retrieves all available credentials from the environment.
// Credential failures are non-fatal: each provider is attempted independently,
// errors are logged and skipped. Returns an error only if no credentials are found at all.
func RetrieveAll() ([]Credential, error) {
	var credentials []Credential

	// Discover AWS credentials
	awsCreds, err := retrieveAWSCredentials()
	if err != nil {
		utils.ErrorLogger.Printf("%s Error retrieving AWS credentials: %v", utils.Iso8601Time(), err)
	} else if awsCreds != nil {
		logCredentialDiscovery("AWS", awsCreds)
		credentials = append(credentials, *awsCreds)
	}

	// Discover Azure credentials
	azureCreds, err := retrieveAzureCredentials()
	if err != nil {
		utils.ErrorLogger.Printf("%s Error retrieving Azure credentials: %v", utils.Iso8601Time(), err)
	} else if azureCreds != nil {
		logCredentialDiscovery("Azure", azureCreds)
		credentials = append(credentials, *azureCreds)
	}

	// Discover GCP credentials
	gcpCreds, err := retrieveGCPCredentials()
	if err != nil {
		utils.ErrorLogger.Printf("%s Error retrieving GCP credentials: %v", utils.Iso8601Time(), err)
	} else if gcpCreds != nil {
		logCredentialDiscovery("GCP", gcpCreds)
		credentials = append(credentials, *gcpCreds)
	}

	// Discover Linode credentials
	linodeCreds, err := retrieveLinodeCredentials()
	if err != nil {
		utils.ErrorLogger.Printf("%s Error retrieving Linode credentials: %v", utils.Iso8601Time(), err)
	} else if linodeCreds != nil {
		logCredentialDiscovery("Linode", linodeCreds)
		credentials = append(credentials, *linodeCreds)
	}

	if len(credentials) == 0 {
		return nil, errors.New("no credentials found")
	}

	return credentials, nil
}

// RetrieveSelected retrieves credentials for the specified providers.
// All selected providers are required: if any provider fails or is not found,
// an error is returned immediately. Use this when the user has explicitly chosen providers.
func RetrieveSelected(selectedProviders []string) ([]Credential, error) {
	var creds []Credential

	for _, provider := range selectedProviders {
		var cred *Credential
		var err error

		switch provider {
		case "AWS":
			cred, err = retrieveAWSCredentials()
		case "Azure":
			cred, err = retrieveAzureCredentials()
		case "GCP":
			cred, err = retrieveGCPCredentials()
		case "Linode":
			cred, err = retrieveLinodeCredentials()
		default:
			return nil, fmt.Errorf("unsupported provider: %s", provider)
		}

		if err != nil {
			return nil, fmt.Errorf("error retrieving %s credentials: %v", provider, err)
		}
		if cred == nil {
			return nil, fmt.Errorf("%s credentials not found", provider)
		}
		creds = append(creds, *cred)
	}

	return creds, nil
}
