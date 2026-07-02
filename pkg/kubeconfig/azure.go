package kubeconfig

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"kubectm/pkg/credentials"
	"kubectm/pkg/utils"

	"github.com/fatih/color"
)

// azureLoginBaseURL and azureManagementBaseURL are the Microsoft Entra ID and
// Azure Resource Manager endpoints. They are variables so tests can point
// them at mock servers.
// API documentation: https://learn.microsoft.com/en-us/rest/api/aks/managed-clusters
// Endpoints used:
//   - POST {login}/{tenant}/oauth2/v2.0/token - Client credentials token grant
//   - GET  /subscriptions/{sub}/providers/Microsoft.ContainerService/managedClusters - List AKS clusters
//   - POST {clusterId}/listClusterUserCredential - Get cluster kubeconfig (base64 encoded)
var (
	azureLoginBaseURL      = "https://login.microsoftonline.com"
	azureManagementBaseURL = "https://management.azure.com"
)

const (
	azureAPIVersion      = "2024-05-01"
	azureDownloadTimeout = 30 * time.Second
)

// aksCluster models the fields kubectm needs from the ARM managedClusters response.
type aksCluster struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Location string `json:"location"`
}

type aksClustersResponse struct {
	Value    []aksCluster `json:"value"`
	NextLink string       `json:"nextLink"`
}

// downloadAzureKubeConfig downloads kubeconfigs for all AKS clusters in the
// subscription associated with the discovered service principal. Per-cluster
// errors are logged and skipped so one bad cluster does not fail the whole
// provider.
func downloadAzureKubeConfig(cred credentials.Credential) error {
	ctx, cancel := context.WithTimeout(context.Background(), azureDownloadTimeout)
	defer cancel()

	token, clusters, err := listAzureClustersWithToken(ctx, cred)
	if err != nil {
		return err
	}

	if len(clusters) == 0 {
		utils.InfoLogger.Printf("%s No AKS clusters found in subscription %s", utils.Iso8601Time(), utils.ObfuscateCredential(cred.Details["SubscriptionID"]))
		return nil
	}

	utils.InfoLogger.Printf("%s Found %d AKS cluster(s)", utils.Iso8601Time(), len(clusters))

	for _, cluster := range clusters {
		if err := processAKSCluster(ctx, token, cluster); err != nil {
			utils.WarnLogger.Printf("%s Failed to process AKS cluster %s: %v", utils.Iso8601Time(), cluster.Name, err)
			continue
		}
	}

	return nil
}

// listAzureClustersWithToken authenticates the service principal and lists
// all AKS clusters in the subscription. The access token is returned so
// callers can reuse it for per-cluster credential requests.
func listAzureClustersWithToken(ctx context.Context, cred credentials.Credential) (string, []aksCluster, error) {
	clientID := cred.Details["ClientID"]
	clientSecret := cred.Details["ClientSecret"]
	tenantID := cred.Details["TenantID"]
	subscriptionID := cred.Details["SubscriptionID"]
	if clientID == "" || clientSecret == "" || tenantID == "" || subscriptionID == "" {
		return "", nil, fmt.Errorf("Azure credentials are incomplete")
	}

	token, err := getAzureAccessToken(ctx, tenantID, clientID, clientSecret)
	if err != nil {
		return "", nil, fmt.Errorf("failed to obtain Azure access token: %v", err)
	}

	clusters, err := listAKSClusters(ctx, token, subscriptionID)
	if err != nil {
		return "", nil, err
	}

	return token, clusters, nil
}

// getAzureAccessToken performs the OAuth2 client credentials grant against
// Microsoft Entra ID for the Azure Resource Manager scope.
func getAzureAccessToken(ctx context.Context, tenantID, clientID, clientSecret string) (string, error) {
	tokenURL := fmt.Sprintf("%s/%s/oauth2/v2.0/token", azureLoginBaseURL, url.PathEscape(tenantID))

	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"scope":         {azureManagementBaseURL + "/.default"},
	}
	return requestAccessToken(ctx, tokenURL, form)
}

// listAKSClusters lists all AKS clusters in the subscription via the Azure
// Resource Manager API, following pagination links.
func listAKSClusters(ctx context.Context, token, subscriptionID string) ([]aksCluster, error) {
	endpoint := fmt.Sprintf("%s/subscriptions/%s/providers/Microsoft.ContainerService/managedClusters?api-version=%s",
		azureManagementBaseURL, url.PathEscape(subscriptionID), azureAPIVersion)

	var allClusters []aksCluster
	for endpoint != "" {
		// Follow only pagination links that stay on the management endpoint,
		// since nextLink comes from the API response and is untrusted.
		if !strings.HasPrefix(endpoint, azureManagementBaseURL+"/") {
			return nil, fmt.Errorf("refusing to follow pagination link outside management endpoint: %s", endpoint)
		}

		page, err := getAKSClustersPage(ctx, token, endpoint)
		if err != nil {
			return nil, err
		}
		allClusters = append(allClusters, page.Value...)
		endpoint = page.NextLink
	}

	return allClusters, nil
}

// getAKSClustersPage fetches a single page of the managedClusters listing.
func getAKSClustersPage(ctx context.Context, token, endpoint string) (*aksClustersResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list AKS clusters, status: %d, body: %s", resp.StatusCode, string(body))
	}

	var page aksClustersResponse
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, err
	}
	return &page, nil
}

// processAKSCluster fetches the user kubeconfig for a single AKS cluster and
// saves it as ~/.kube/{cluster-name}@{resource-group}-kubeconfig.yaml.
func processAKSCluster(ctx context.Context, token string, cluster aksCluster) error {
	resourceGroup := resourceGroupFromID(cluster.ID)

	// Cluster name and resource group come from the ARM API response.
	// Validate them against a strict allowlist before they are used to build
	// a file path.
	if !isValidAzureIdentifier(cluster.Name) {
		return fmt.Errorf("cluster %q has an unexpected name format", cluster.Name)
	}
	if !isValidAzureIdentifier(resourceGroup) {
		return fmt.Errorf("cluster %s has an unexpected resource group format %q", cluster.Name, resourceGroup)
	}

	contextName := fmt.Sprintf("%s@%s", cluster.Name, resourceGroup)
	utils.ActionLogger.Printf("%s Downloading kubeconfig for AKS cluster: %s",
		utils.Iso8601Time(), color.New(color.Bold).Sprint(contextName))

	kubeconfig, err := getAKSClusterKubeconfig(ctx, token, cluster.ID)
	if err != nil {
		return err
	}

	return saveKubeconfigToFile(contextName, kubeconfig)
}

// getAKSClusterKubeconfig calls listClusterUserCredential for the cluster and
// returns the decoded kubeconfig content.
func getAKSClusterKubeconfig(ctx context.Context, token, clusterID string) (string, error) {
	// The cluster ID is an ARM resource path from the list response; ensure it
	// looks like one before interpolating it into a URL.
	if !strings.HasPrefix(clusterID, "/subscriptions/") {
		return "", fmt.Errorf("unexpected cluster resource ID format: %s", clusterID)
	}

	endpoint := fmt.Sprintf("%s%s/listClusterUserCredential?api-version=%s", azureManagementBaseURL, clusterID, azureAPIVersion)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get cluster credentials, status: %d, body: %s", resp.StatusCode, string(body))
	}

	var credentialResponse struct {
		Kubeconfigs []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"kubeconfigs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&credentialResponse); err != nil {
		return "", err
	}

	for _, kc := range credentialResponse.Kubeconfigs {
		if kc.Name != "clusterUser" {
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(kc.Value)
		if err != nil {
			return "", fmt.Errorf("failed to decode kubeconfig: %v", err)
		}
		return string(decoded), nil
	}

	return "", fmt.Errorf("credential response did not contain a clusterUser kubeconfig")
}

// resourceGroupFromID extracts the resource group name from an ARM resource
// ID like /subscriptions/{sub}/resourcegroups/{rg}/providers/... The segment
// name is matched case-insensitively because ARM is inconsistent about it.
func resourceGroupFromID(resourceID string) string {
	segments := strings.Split(resourceID, "/")
	for i, segment := range segments {
		if strings.EqualFold(segment, "resourcegroups") && i+1 < len(segments) {
			return segments[i+1]
		}
	}
	return ""
}

// azureIdentifierPattern matches the characters allowed in AKS cluster names
// and resource group names. Resource groups additionally allow parentheses.
// The set excludes whitespace, YAML metacharacters and path separators.
var azureIdentifierPattern = regexp.MustCompile(`^[A-Za-z0-9._()-]+$`)

// isValidAzureIdentifier reports whether s is a safe AKS cluster name or
// resource group name.
func isValidAzureIdentifier(s string) bool {
	return s != "" && azureIdentifierPattern.MatchString(s)
}
