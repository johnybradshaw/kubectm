package kubeconfig

import (
    "kubectm/pkg/credentials"
    "fmt"
)

// DownloadConfigs downloads the kubeconfig files from the specified providers.
// It loops through the given credentials and uses the provider to download the
// corresponding kubeconfig file.
func DownloadConfigs(creds []credentials.Credential) error {
    for _, cred := range creds {
        switch cred.Provider {
        case "Linode":
            // Download the Linode cluster configuration
            err := downloadLinodeKubeConfig(cred)
            if err != nil {
                return fmt.Errorf("error downloading Linode kubeconfig: %v", err)
            }
        // Add cases for other providers like AWS, Azure, GCP, etc.
        default:
            // Print a message to the user if the provider is not supported
            fmt.Printf("Provider %s is not supported yet\n", cred.Provider)
        }
    }
    return nil
}
