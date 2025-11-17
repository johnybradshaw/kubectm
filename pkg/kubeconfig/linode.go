package kubeconfig

import (
    "encoding/base64"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
    "path/filepath"
    "github.com/fatih/color"
    "kubectm/pkg/credentials"
    "kubectm/pkg/utils"  // Import the utils package
)

// linodeAPIBaseURL is the base URL for the Linode API v4.
// API Documentation: https://techdocs.akamai.com/linode-api/reference/api
// LKE endpoints used:
//   - GET /lke/clusters - List all LKE clusters
//   - GET /lke/clusters/{clusterId}/kubeconfig - Get cluster kubeconfig (base64 encoded)
const linodeAPIBaseURL = "https://api.linode.com/v4"

type LinodeCluster struct {
    ID    int    `json:"id"`
    Label string `json:"label"`
}

type LinodeClustersResponse struct {
    Data    []LinodeCluster `json:"data"`
    Page    int             `json:"page"`
    Pages   int             `json:"pages"`
    Results int             `json:"results"`
}

type KubeconfigResponse struct {
    Kubeconfig string `json:"kubeconfig"`
}

// downloadLinodeKubeConfig downloads the Linode cluster configuration.
// It takes a single Credential object as an argument which contains the
// access token and provider information.
// Returns an error if the download or saving process fails.
func downloadLinodeKubeConfig(cred credentials.Credential) error {
    // Get the access token from the credential details
    token := cred.Details["AccessToken"]
    if token == "" {
        return fmt.Errorf("Linode access token is missing")
    }

    // Retrieve the list of Linode clusters
    clusters, err := getLinodeClusters(token)
    if err != nil {
        return fmt.Errorf("failed to retrieve Linode clusters: %v", err)
    }

    // Loop through each cluster and download the kubeconfig
    // If the download or saving process fails for any cluster, stop
    // the loop and return an error.
    for _, cluster := range clusters { 
        utils.ActionLogger.Printf("%s Downloading kubeconfig for cluster: %s", utils.Iso8601Time(), color.New(color.Bold).Sprint(cluster.Label))
        kubeconfig, err := getLinodeKubeconfig(token, cluster.ID)
        if err != nil {
            return fmt.Errorf("failed to retrieve kubeconfig for cluster %s: %v", cluster.Label, err)
        }

        err = saveKubeconfigToFile(cluster.Label, kubeconfig)
        if err != nil {
            return fmt.Errorf("failed to save kubeconfig for cluster %s: %v", cluster.Label, err)
        }
    }

    return nil
}

// getLinodeClusters retrieves the list of Linode clusters using the
// specified token. The token is used to authenticate the request.
//
// It returns a slice of LinodeCluster objects and an error if the request
// fails.
func getLinodeClusters(token string) ([]LinodeCluster, error) {
    // Construct the URL for the Linode API.
    url := fmt.Sprintf("%s/lke/clusters", linodeAPIBaseURL)

    // Create a new HTTP request.
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        // If the request creation fails, return the error.
        return nil, err
    }

    // Set the Authorization header with the token.
    req.Header.Set("Authorization", "Bearer "+token)
    // Set the Content-Type header to application/json.
    req.Header.Set("Content-Type", "application/json")

    // Create a new HTTP client.
    client := &http.Client{}

    // Send the request and get the response.
    resp, err := client.Do(req)
    if err != nil {
        // If the request fails, return the error.
        return nil, err
    }
    defer resp.Body.Close()

    // Check the status code of the response.
    if resp.StatusCode != http.StatusOK {
        // If the status code is not 200, read the body and return an error.
        body, _ := ioutil.ReadAll(resp.Body)
        return nil, fmt.Errorf("failed to list clusters, status: %d, body: %s", resp.StatusCode, string(body))
    }

    // Decode the response body into a LinodeClustersResponse object.
    var clustersResponse LinodeClustersResponse
    err = json.NewDecoder(resp.Body).Decode(&clustersResponse)
    if err != nil {
        // If the decoding fails, return the error.
        return nil, err
    }

    // Return the slice of LinodeCluster objects.
    return clustersResponse.Data, nil
}

// getLinodeKubeconfig retrieves the kubeconfig file for the specified Linode
// cluster using the given token. It sends a GET request to the Linode API and
// returns the decoded kubeconfig file as a string.
func getLinodeKubeconfig(token string, clusterID int) (string, error) {
    url := fmt.Sprintf("%s/lke/clusters/%d/kubeconfig", linodeAPIBaseURL, clusterID)
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        // If the request creation fails, return the error.
        return "", err
    }

    // Set the Authorization header with the token.
    req.Header.Set("Authorization", "Bearer "+token)
    // Set the Content-Type header to application/json.
    req.Header.Set("Content-Type", "application/json")

    // Create a new HTTP client.
    client := &http.Client{}

    // Send the request and get the response.
    resp, err := client.Do(req)
    if err != nil {
        // If the request fails, return the error.
        return "", err
    }
    defer resp.Body.Close()

    // Check the status code of the response.
    if resp.StatusCode != http.StatusOK {
        // If the status code is not 200, read the body and return an error.
        body, _ := ioutil.ReadAll(resp.Body)
        return "", fmt.Errorf("failed to retrieve kubeconfig, status: %d, body: %s", resp.StatusCode, string(body))
    }

    // Decode the response body into a KubeconfigResponse object.
    var kubeconfigResponse KubeconfigResponse
    err = json.NewDecoder(resp.Body).Decode(&kubeconfigResponse)
    if err != nil {
        // If the decoding fails, return the error.
        return "", err
    }

    // Decode the base64 encoded kubeconfig file.
    decodedKubeconfig, err := base64.StdEncoding.DecodeString(kubeconfigResponse.Kubeconfig)
    if err != nil {
        // If the decoding fails, return the error.
        return "", fmt.Errorf("failed to decode kubeconfig for cluster %d: %v", clusterID, err)
    }

    // Return the decoded kubeconfig file as a string.
    return string(decodedKubeconfig), nil
}

// saveKubeconfigToFile saves the given kubeconfig string to a file in the
// ~/.kube directory. The file name is determined by the clusterLabel
// argument. If the file already exists, it is overwritten.
//
// The function returns an error if there is a problem writing the file.
func saveKubeconfigToFile(clusterLabel string, kubeconfig string) error {
    // Get the user's home directory
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return err
    }

    // Create the ~/.kube directory if it doesn't exist
    kubeconfigDir := filepath.Join(homeDir, ".kube")
    err = os.MkdirAll(kubeconfigDir, os.ModePerm)
    if err != nil {
        return err
    }

    // Create the file name by appending the cluster label to the
    // "kubeconfig.yaml" string
    kubeconfigFile := filepath.Join(kubeconfigDir, fmt.Sprintf("%s-kubeconfig.yaml", clusterLabel))

    // Write the kubeconfig string to the file
    err = os.WriteFile(kubeconfigFile, []byte(kubeconfig), 0600)
    if err != nil {
        return err
    }

    // Log a message to the user indicating that the file was saved
    utils.InfoLogger.Printf("%s Kubeconfig saved to %s", utils.Iso8601Time(), kubeconfigFile)
    return nil
}
