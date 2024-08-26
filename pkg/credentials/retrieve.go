package credentials

import (
    "errors"   // Import the errors package
    "log"
    "kubectm/pkg/utils"
)

type Credential struct {
    Provider string
    Details  map[string]string
}

// RetrieveAll retrieves all available credentials from the environment.
func RetrieveAll() ([]Credential, error) {
    var credentials []Credential

    // Discover AWS credentials
    awsCreds, err := retrieveAWSCredentials()  // Stub function
    if err != nil {
        log.Printf("Error retrieving AWS credentials: %v", err)
    } else if awsCreds != nil {
        // Obfuscate credentials before logging
        obfuscatedCreds := map[string]string{
            "AccessKey": utils.ObfuscateCredential(awsCreds.Details["AccessKey"]),
            "SecretKey": utils.ObfuscateCredential(awsCreds.Details["SecretKey"]),
        }
        log.Printf("AWS credentials found: %v", obfuscatedCreds)
        credentials = append(credentials, *awsCreds)
    }

    // Discover Azure credentials (assume similar to AWS)
    azureCreds, err := retrieveAzureCredentials()  // Stub function
    if err != nil {
        log.Printf("Error retrieving Azure credentials: %v", err)
    } else if azureCreds != nil {
        obfuscatedCreds := map[string]string{
            "ClientID":     utils.ObfuscateCredential(azureCreds.Details["ClientID"]),
            "ClientSecret": utils.ObfuscateCredential(azureCreds.Details["ClientSecret"]),
        }
        log.Printf("Azure credentials found: %v", obfuscatedCreds)
        credentials = append(credentials, *azureCreds)
    }

    // Discover GCP credentials (assume similar to AWS)
    gcpCreds, err := retrieveGCPCredentials()  // Stub function
    if err != nil {
        log.Printf("Error retrieving GCP credentials: %v", err)
    } else if gcpCreds != nil {
        obfuscatedCreds := map[string]string{
            "ProjectID": utils.ObfuscateCredential(gcpCreds.Details["ProjectID"]),
        }
        log.Printf("GCP credentials found: %v", obfuscatedCreds)
        credentials = append(credentials, *gcpCreds)
    }

    // Discover Linode credentials
    linodeCreds, err := retrieveLinodeCredentials()
    if err != nil {
        log.Printf("Error retrieving Linode credentials: %v", err)
    } else if linodeCreds != nil {
        obfuscatedCreds := map[string]string{
            "AccessToken": utils.ObfuscateCredential(linodeCreds.Details["AccessToken"]),
        }
        log.Printf("Linode credentials found: %v", obfuscatedCreds)
        credentials = append(credentials, *linodeCreds)
    }

    if len(credentials) == 0 {
        return nil, errors.New("no credentials found")
    }

    return credentials, nil
}
