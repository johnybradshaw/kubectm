package kubeconfig

import (
    "encoding/base64"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "path/filepath"

    "kubectm/pkg/credentials"
)

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

// downloadLinodeKubeConfig downloads the kubeconfig for each cluster in Linode
//
// This function takes a single parameter: a Credential object containing the
// Linode access token.
// It retrieves the list of clusters from Linode, downloads the kubeconfig for
// each cluster, and saves the kubeconfig to a file in the .kube directory.
func downloadLinodeKubeConfig(cred credentials.Credential) error {
    token := cred.Details["AccessToken"]
    if token == "" {
        return fmt.Errorf("Linode access token is missing")
    }

    // Step 1: Retrieve the list of clusters
    clusters, err := getLinodeClusters(token)
    if err != nil {
        return fmt.Errorf("failed to retrieve Linode clusters: %v", err)
    }

    // Step 2: Download the kubeconfig for each cluster
    for _, cluster := range clusters {
        log.Printf("Downloading kubeconfig for cluster: %s", cluster.Label)
        kubeconfig, err := getLinodeKubeconfig(token, cluster.ID)
        if err != nil {
            return fmt.Errorf("failed to retrieve kubeconfig for cluster %s: %v", cluster.Label, err)
        }

        // Step 3: Save the kubeconfig to a file
        err = saveKubeconfigToFile(cluster.Label, kubeconfig)
        if err != nil {
            return fmt.Errorf("failed to save kubeconfig for cluster %s: %v", cluster.Label, err)
        }
    }

    return nil
}

// getLinodeClusters retrieves the list of Kubernetes clusters from Linode
//
// This function takes a single parameter: the Linode access token.
// It returns a slice of LinodeCluster objects and an error if any occurs.
func getLinodeClusters(token string) ([]LinodeCluster, error) {
    // Construct the URL to retrieve the clusters
    url := fmt.Sprintf("%s/lke/clusters", linodeAPIBaseURL)

    // Create an HTTP request to retrieve the clusters
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }

    // Set the required headers for the request
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Content-Type", "application/json")

    // Create an HTTP client and send the request
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // Check the status code of the response
    if resp.StatusCode != http.StatusOK {
        // Read the response body if the status code is not 200
        body, _ := ioutil.ReadAll(resp.Body)
        return nil, fmt.Errorf("failed to list clusters, status: %d, body: %s", resp.StatusCode, string(body))
    }

    // Decode the response body into a LinodeClustersResponse object
    var clustersResponse LinodeClustersResponse
    err = json.NewDecoder(resp.Body).Decode(&clustersResponse)
    if err != nil {
        return nil, err
    }

    // Return the list of clusters
    return clustersResponse.Data, nil
}

// getLinodeKubeconfig retrieves and decodes the kubeconfig for a given cluster ID
//
// This function takes two parameters: the Linode access token and the cluster ID.
// It returns the decoded kubeconfig content as a string and an error if any occurs.
func getLinodeKubeconfig(token string, clusterID int) (string, error) {
    // Construct the URL to retrieve the kubeconfig
    url := fmt.Sprintf("%s/lke/clusters/%d/kubeconfig", linodeAPIBaseURL, clusterID)

    // Create an HTTP request to retrieve the kubeconfig
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return "", err
    }

    // Add the required headers to the request
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Content-Type", "application/json")

    // Create an HTTP client and send the request
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    // Check the status code of the response
    if resp.StatusCode != http.StatusOK {
        body, _ := ioutil.ReadAll(resp.Body)
        return "", fmt.Errorf("failed to retrieve kubeconfig, status: %d, body: %s", resp.StatusCode, string(body))
    }

    // Decode the JSON response
    var kubeconfigResponse KubeconfigResponse
    err = json.NewDecoder(resp.Body).Decode(&kubeconfigResponse)
    if err != nil {
        return "", err
    }

    // Decode the base64 encoded kubeconfig
    decodedKubeconfig, err := base64.StdEncoding.DecodeString(kubeconfigResponse.Kubeconfig)
    if err != nil {
        return "", fmt.Errorf("failed to decode kubeconfig for cluster %d: %v", clusterID, err)
    }

    // Return the decoded kubeconfig content
    return string(decodedKubeconfig), nil
}

// saveKubeconfigToFile saves the decoded kubeconfig content to a file
//
// This function takes two parameters: the cluster label and the decoded kubeconfig content.
// It saves the kubeconfig content to a file named <clusterLabel>-kubeconfig.yaml in the default
// Kubernetes configuration directory.
func saveKubeconfigToFile(clusterLabel string, kubeconfig string) error {
    // Get the user's home directory
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return err
    }

    // Create the default Kubernetes configuration directory if it doesn't exist
    kubeconfigDir := filepath.Join(homeDir, ".kube")
    err = os.MkdirAll(kubeconfigDir, os.ModePerm)
    if err != nil {
        return err
    }

    // Create the kubeconfig file
    kubeconfigFile := filepath.Join(kubeconfigDir, fmt.Sprintf("%s-kubeconfig.yaml", clusterLabel))
    err = os.WriteFile(kubeconfigFile, []byte(kubeconfig), 0600)
    if err != nil {
        return err
    }

    log.Printf("Kubeconfig saved to %s", kubeconfigFile)
    return nil
}
