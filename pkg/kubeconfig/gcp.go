package kubeconfig

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"
	"time"

	"kubectm/pkg/credentials"
	"kubectm/pkg/utils"

	"github.com/fatih/color"
)

// gkeAPIBaseURL is the base URL for the GKE API. It is a variable so tests
// can point it at a mock server.
// API documentation: https://cloud.google.com/kubernetes-engine/docs/reference/rest
// Endpoint used:
//   - GET /projects/{projectID}/locations/-/clusters - List clusters across all locations
var gkeAPIBaseURL = "https://container.googleapis.com/v1"

// googleTokenURL is the default OAuth2 token endpoint used when the
// credentials JSON does not specify its own token_uri.
var googleTokenURL = "https://oauth2.googleapis.com/token"

const (
	gcpDownloadTimeout = 30 * time.Second
	gcpCloudScope      = "https://www.googleapis.com/auth/cloud-platform"
)

// gcpAuthJSON models the credential fields needed to obtain an OAuth2 access
// token from either a service account key or gcloud application default
// credentials (authorized_user).
type gcpAuthJSON struct {
	Type         string `json:"type"`
	ClientEmail  string `json:"client_email"`
	PrivateKey   string `json:"private_key"`
	TokenURI     string `json:"token_uri"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
}

// gkeCluster models the fields kubectm needs from the GKE clusters.list response.
type gkeCluster struct {
	Name       string `json:"name"`
	Location   string `json:"location"`
	Endpoint   string `json:"endpoint"`
	Status     string `json:"status"`
	MasterAuth struct {
		ClusterCACertificate string `json:"clusterCaCertificate"`
	} `json:"masterAuth"`
}

type gkeClustersResponse struct {
	Clusters []gkeCluster `json:"clusters"`
}

// downloadGCPKubeConfig downloads kubeconfigs for all GKE clusters in the
// project associated with the discovered credentials. Per-cluster errors are
// logged and skipped so one bad cluster does not fail the whole provider.
func downloadGCPKubeConfig(cred credentials.Credential) error {
	ctx, cancel := context.WithTimeout(context.Background(), gcpDownloadTimeout)
	defer cancel()

	clusters, err := listGCPClusters(ctx, cred)
	if err != nil {
		return err
	}

	if len(clusters) == 0 {
		utils.InfoLogger.Printf("%s No GKE clusters found in project %s", utils.Iso8601Time(), cred.Details["ProjectID"])
		return nil
	}

	utils.InfoLogger.Printf("%s Found %d GKE cluster(s) in project %s", utils.Iso8601Time(), len(clusters), cred.Details["ProjectID"])

	for _, cluster := range clusters {
		if err := processGKECluster(cluster); err != nil {
			utils.WarnLogger.Printf("%s Failed to process GKE cluster %s: %v", utils.Iso8601Time(), cluster.Name, err)
			continue
		}
	}

	return nil
}

// listGCPClusters authenticates with the discovered credentials and lists all
// GKE clusters in the configured project across all locations.
func listGCPClusters(ctx context.Context, cred credentials.Credential) ([]gkeCluster, error) {
	credsFile := cred.Details["CredentialsFile"]
	projectID := cred.Details["ProjectID"]
	if credsFile == "" {
		return nil, fmt.Errorf("GCP credentials file is missing")
	}
	if projectID == "" {
		return nil, fmt.Errorf("GCP project ID is missing")
	}

	token, err := getGCPAccessToken(ctx, credsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain GCP access token: %v", err)
	}

	return listGKEClusters(ctx, token, projectID)
}

// listGKEClusters lists GKE clusters in the given project across all
// locations using the GKE REST API.
func listGKEClusters(ctx context.Context, token, projectID string) ([]gkeCluster, error) {
	endpoint := fmt.Sprintf("%s/projects/%s/locations/-/clusters", gkeAPIBaseURL, url.PathEscape(projectID))

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
		return nil, fmt.Errorf("failed to list GKE clusters, status: %d, body: %s", resp.StatusCode, string(body))
	}

	var clustersResponse gkeClustersResponse
	if err := json.NewDecoder(resp.Body).Decode(&clustersResponse); err != nil {
		return nil, err
	}

	return clustersResponse.Clusters, nil
}

// processGKECluster validates a single GKE cluster's metadata and generates +
// saves an exec-based kubeconfig for it.
func processGKECluster(cluster gkeCluster) error {
	// Cluster name and location come from the GKE API response. Validate them
	// against a strict allowlist before they are interpolated into the
	// kubeconfig template or used to build a file path.
	if !isValidEKSIdentifier(cluster.Name) {
		return fmt.Errorf("cluster %q has an unexpected name format", cluster.Name)
	}
	if !isValidEKSIdentifier(cluster.Location) {
		return fmt.Errorf("cluster %s has an unexpected location format %q", cluster.Name, cluster.Location)
	}
	if cluster.Endpoint == "" || cluster.MasterAuth.ClusterCACertificate == "" {
		return fmt.Errorf("cluster %s has incomplete data", cluster.Name)
	}

	contextName := fmt.Sprintf("%s@%s", cluster.Name, cluster.Location)
	utils.ActionLogger.Printf("%s Downloading kubeconfig for GKE cluster: %s",
		utils.Iso8601Time(), color.New(color.Bold).Sprint(contextName))

	kubeconfigContent := generateGKEKubeconfig(cluster)
	if kubeconfigContent == "" {
		return fmt.Errorf("failed to generate kubeconfig for cluster %s", cluster.Name)
	}

	return saveKubeconfigToFile(contextName, kubeconfigContent)
}

// gkeKubeconfigTemplate is the template for generating GKE kubeconfig files.
// Authentication uses the standard gke-gcloud-auth-plugin exec plugin.
var gkeKubeconfigTemplate = template.Must(template.New("gkekubeconfig").Parse(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://{{.Endpoint}}
    certificate-authority-data: {{.CAData}}
  name: {{.ContextName}}
contexts:
- context:
    cluster: {{.ContextName}}
    user: {{.ContextName}}
  name: {{.ContextName}}
current-context: {{.ContextName}}
users:
- name: {{.ContextName}}
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      command: gke-gcloud-auth-plugin
      installHint: Install gke-gcloud-auth-plugin for kubectl by running gcloud components install gke-gcloud-auth-plugin
      provideClusterInfo: true
`))

// generateGKEKubeconfig generates a kubeconfig YAML string for a GKE cluster
// that uses the gke-gcloud-auth-plugin exec plugin for authentication.
func generateGKEKubeconfig(cluster gkeCluster) string {
	data := struct {
		Endpoint    string
		CAData      string
		ContextName string
	}{
		Endpoint:    cluster.Endpoint,
		CAData:      cluster.MasterAuth.ClusterCACertificate,
		ContextName: fmt.Sprintf("%s@%s", cluster.Name, cluster.Location),
	}

	var buf bytes.Buffer
	if err := gkeKubeconfigTemplate.Execute(&buf, data); err != nil {
		utils.ErrorLogger.Printf("%s Error executing GKE kubeconfig template: %v", utils.Iso8601Time(), err)
		return ""
	}

	return buf.String()
}

// getGCPAccessToken reads a Google credentials JSON file and exchanges it for
// an OAuth2 access token. Service account keys use the signed-JWT grant;
// gcloud application default credentials (authorized_user) use the refresh
// token grant.
func getGCPAccessToken(ctx context.Context, credsFile string) (string, error) {
	data, err := os.ReadFile(credsFile)
	if err != nil {
		return "", fmt.Errorf("error reading credentials file: %v", err)
	}

	var auth gcpAuthJSON
	if err := json.Unmarshal(data, &auth); err != nil {
		return "", fmt.Errorf("error parsing credentials file: %v", err)
	}

	switch auth.Type {
	case "service_account":
		return serviceAccountAccessToken(ctx, auth)
	case "authorized_user":
		return authorizedUserAccessToken(ctx, auth)
	default:
		return "", fmt.Errorf("unsupported GCP credentials type %q", auth.Type)
	}
}

// serviceAccountAccessToken performs the OAuth2 JWT bearer grant for a
// service account key: it signs a JWT assertion with the account's private
// key and exchanges it for an access token at the key's token endpoint.
func serviceAccountAccessToken(ctx context.Context, auth gcpAuthJSON) (string, error) {
	if auth.ClientEmail == "" || auth.PrivateKey == "" {
		return "", fmt.Errorf("service account key is missing client_email or private_key")
	}

	tokenURL := auth.TokenURI
	if tokenURL == "" {
		tokenURL = googleTokenURL
	}

	assertion, err := signServiceAccountJWT(auth, tokenURL, time.Now())
	if err != nil {
		return "", err
	}

	form := url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {assertion},
	}
	return requestAccessToken(ctx, tokenURL, form)
}

// authorizedUserAccessToken performs the OAuth2 refresh token grant for
// gcloud application default credentials.
func authorizedUserAccessToken(ctx context.Context, auth gcpAuthJSON) (string, error) {
	if auth.ClientID == "" || auth.ClientSecret == "" || auth.RefreshToken == "" {
		return "", fmt.Errorf("authorized_user credentials are missing client_id, client_secret or refresh_token")
	}

	tokenURL := auth.TokenURI
	if tokenURL == "" {
		tokenURL = googleTokenURL
	}

	form := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {auth.ClientID},
		"client_secret": {auth.ClientSecret},
		"refresh_token": {auth.RefreshToken},
	}
	return requestAccessToken(ctx, tokenURL, form)
}

// signServiceAccountJWT builds and signs the RS256 JWT assertion used by the
// service account JWT bearer grant.
func signServiceAccountJWT(auth gcpAuthJSON, audience string, now time.Time) (string, error) {
	block, _ := pem.Decode([]byte(auth.PrivateKey))
	if block == nil {
		return "", fmt.Errorf("service account private_key is not valid PEM")
	}

	// Google issues PKCS#8 keys; fall back to PKCS#1 for robustness.
	var rsaKey *rsa.PrivateKey
	if parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		key, ok := parsed.(*rsa.PrivateKey)
		if !ok {
			return "", fmt.Errorf("service account private_key is not an RSA key")
		}
		rsaKey = key
	} else if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		rsaKey = key
	} else {
		return "", fmt.Errorf("failed to parse service account private_key: %v", err)
	}

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	claims, err := json.Marshal(map[string]interface{}{
		"iss":   auth.ClientEmail,
		"scope": gcpCloudScope,
		"aud":   audience,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	})
	if err != nil {
		return "", err
	}

	signingInput := header + "." + base64.RawURLEncoding.EncodeToString(claims)
	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT assertion: %v", err)
	}

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

// requestAccessToken posts an OAuth2 token request and returns the access token.
func requestAccessToken(ctx context.Context, tokenURL string, form url.Values) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request failed, status: %d, body: %s", resp.StatusCode, string(body))
	}

	var tokenResponse struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return "", err
	}
	if tokenResponse.AccessToken == "" {
		return "", fmt.Errorf("token response did not contain an access token")
	}

	return tokenResponse.AccessToken, nil
}
