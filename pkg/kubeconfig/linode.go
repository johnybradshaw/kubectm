package kubeconfig

import (
    "encoding/base64"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
    "path/filepath"
    "kubectm/pkg/credentials"
    "kubectm/pkg/utils"  // Import the utils package
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

func downloadLinodeKubeConfig(cred credentials.Credential) error {
    token := cred.Details["AccessToken"]
    if token == "" {
        return fmt.Errorf("Linode access token is missing")
    }

    clusters, err := getLinodeClusters(token)
    if err != nil {
        return fmt.Errorf("failed to retrieve Linode clusters: %v", err)
    }

    for _, cluster := range clusters {
        utils.ActionLogger.Printf("%s Downloading kubeconfig for cluster: %s", utils.Iso8601Time(), cluster.Label)
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

func getLinodeClusters(token string) ([]LinodeCluster, error) {
    url := fmt.Sprintf("%s/lke/clusters", linodeAPIBaseURL)
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }

    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := ioutil.ReadAll(resp.Body)
        return nil, fmt.Errorf("failed to list clusters, status: %d, body: %s", resp.StatusCode, string(body))
    }

    var clustersResponse LinodeClustersResponse
    err = json.NewDecoder(resp.Body).Decode(&clustersResponse)
    if err != nil {
        return nil, err
    }

    return clustersResponse.Data, nil
}

func getLinodeKubeconfig(token string, clusterID int) (string, error) {
    url := fmt.Sprintf("%s/lke/clusters/%d/kubeconfig", linodeAPIBaseURL, clusterID)
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return "", err
    }

    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := ioutil.ReadAll(resp.Body)
        return "", fmt.Errorf("failed to retrieve kubeconfig, status: %d, body: %s", resp.StatusCode, string(body))
    }

    var kubeconfigResponse KubeconfigResponse
    err = json.NewDecoder(resp.Body).Decode(&kubeconfigResponse)
    if err != nil {
        return "", err
    }

    decodedKubeconfig, err := base64.StdEncoding.DecodeString(kubeconfigResponse.Kubeconfig)
    if err != nil {
        return "", fmt.Errorf("failed to decode kubeconfig for cluster %d: %v", clusterID, err)
    }

    return string(decodedKubeconfig), nil
}

func saveKubeconfigToFile(clusterLabel string, kubeconfig string) error {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return err
    }

    kubeconfigDir := filepath.Join(homeDir, ".kube")
    err = os.MkdirAll(kubeconfigDir, os.ModePerm)
    if err != nil {
        return err
    }

    kubeconfigFile := filepath.Join(kubeconfigDir, fmt.Sprintf("%s-kubeconfig.yaml", clusterLabel))
    err = os.WriteFile(kubeconfigFile, []byte(kubeconfig), 0600)
    if err != nil {
        return err
    }

    utils.InfoLogger.Printf("%s Kubeconfig saved to %s", utils.Iso8601Time(), kubeconfigFile)
    return nil
}