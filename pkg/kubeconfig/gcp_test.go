package kubeconfig

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kubectm/pkg/credentials"
)

// testServiceAccountJSON builds a service account key JSON with a freshly
// generated RSA key. Token requests go to googleTokenURL, which
// newTokenServer points at a mock server.
func testServiceAccountJSON(t *testing.T) string {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("failed to marshal private key: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	sa := map[string]string{
		"type":         "service_account",
		"project_id":   "test-project",
		"client_email": "svc@test-project.iam.gserviceaccount.com",
		"private_key":  string(keyPEM),
	}
	data, err := json.Marshal(sa)
	if err != nil {
		t.Fatalf("failed to marshal service account JSON: %v", err)
	}
	return string(data)
}

// writeCredsFile writes a credentials JSON to a temp file and returns its path.
func writeCredsFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "creds.json")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write credentials file: %v", err)
	}
	return path
}

// newTokenServer returns an httptest server that validates the expected grant
// type and responds with an access token, and points googleTokenURL at it for
// the duration of the test.
func newTokenServer(t *testing.T, wantGrantType string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("failed to parse token request form: %v", err)
		}
		if got := r.PostForm.Get("grant_type"); got != wantGrantType {
			t.Errorf("expected grant_type %q, got %q", wantGrantType, got)
		}
		if wantGrantType == "urn:ietf:params:oauth:grant-type:jwt-bearer" {
			assertion := r.PostForm.Get("assertion")
			if len(strings.Split(assertion, ".")) != 3 {
				t.Errorf("expected a three-part JWT assertion, got %q", assertion)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token": "test-access-token", "token_type": "Bearer"}`)
	}))
	origTokenURL := googleTokenURL
	googleTokenURL = server.URL
	t.Cleanup(func() { googleTokenURL = origTokenURL })
	return server
}

func TestGetGCPAccessTokenServiceAccount(t *testing.T) {
	server := newTokenServer(t, "urn:ietf:params:oauth:grant-type:jwt-bearer")
	defer server.Close()

	credsPath := writeCredsFile(t, testServiceAccountJSON(t))

	token, err := getGCPAccessToken(context.Background(), credsPath)
	if err != nil {
		t.Fatalf("getGCPAccessToken() error = %v", err)
	}
	if token != "test-access-token" {
		t.Errorf("expected token test-access-token, got %q", token)
	}
}

func TestGetGCPAccessTokenAuthorizedUser(t *testing.T) {
	server := newTokenServer(t, "refresh_token")
	defer server.Close()

	credsPath := writeCredsFile(t, `{
		"type": "authorized_user",
		"client_id": "id",
		"client_secret": "secret",
		"refresh_token": "refresh"
	}`)

	token, err := getGCPAccessToken(context.Background(), credsPath)
	if err != nil {
		t.Fatalf("getGCPAccessToken() error = %v", err)
	}
	if token != "test-access-token" {
		t.Errorf("expected token test-access-token, got %q", token)
	}
}

func TestGetGCPAccessTokenUnsupportedType(t *testing.T) {
	credsPath := writeCredsFile(t, `{"type": "external_account"}`)

	if _, err := getGCPAccessToken(context.Background(), credsPath); err == nil {
		t.Fatal("expected error for unsupported credentials type, got nil")
	}
}

func TestGetGCPAccessTokenIncompleteServiceAccount(t *testing.T) {
	credsPath := writeCredsFile(t, `{"type": "service_account", "client_email": "a@b.c"}`)

	if _, err := getGCPAccessToken(context.Background(), credsPath); err == nil {
		t.Fatal("expected error for incomplete service account key, got nil")
	}
}

func TestListGKEClusters(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectedError bool
		expectedCount int
	}{
		{
			name:       "successful listing",
			statusCode: http.StatusOK,
			responseBody: `{"clusters": [
				{"name": "prod", "location": "us-central1", "endpoint": "1.2.3.4", "masterAuth": {"clusterCaCertificate": "Y2E="}},
				{"name": "dev", "location": "europe-west1-b", "endpoint": "5.6.7.8", "masterAuth": {"clusterCaCertificate": "Y2E="}}
			]}`,
			expectedCount: 2,
		},
		{
			name:          "no clusters",
			statusCode:    http.StatusOK,
			responseBody:  `{}`,
			expectedCount: 0,
		},
		{
			name:          "unauthorized",
			statusCode:    http.StatusUnauthorized,
			responseBody:  `{"error": {"message": "invalid token"}}`,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
					t.Errorf("expected bearer token header, got %q", got)
				}
				if want := "/projects/test-project/locations/-/clusters"; r.URL.Path != want {
					t.Errorf("expected path %s, got %s", want, r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
				fmt.Fprint(w, tt.responseBody)
			}))
			defer server.Close()

			origBase := gkeAPIBaseURL
			gkeAPIBaseURL = server.URL
			defer func() { gkeAPIBaseURL = origBase }()

			clusters, err := listGKEClusters(context.Background(), "test-token", "test-project")
			if tt.expectedError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("listGKEClusters() error = %v", err)
			}
			if len(clusters) != tt.expectedCount {
				t.Errorf("expected %d clusters, got %d", tt.expectedCount, len(clusters))
			}
		})
	}
}

func TestGenerateGKEKubeconfig(t *testing.T) {
	cluster := gkeCluster{
		Name:     "prod",
		Location: "us-central1",
		Endpoint: "1.2.3.4",
	}
	cluster.MasterAuth.ClusterCACertificate = "Y2EtZGF0YQ=="

	content := generateGKEKubeconfig(cluster)

	for _, want := range []string{
		"server: https://1.2.3.4",
		"certificate-authority-data: Y2EtZGF0YQ==",
		"name: prod@us-central1",
		"current-context: prod@us-central1",
		"command: gke-gcloud-auth-plugin",
		"provideClusterInfo: true",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("expected kubeconfig to contain %q, got:\n%s", want, content)
		}
	}
}

func TestProcessGKEClusterInvalidMetadata(t *testing.T) {
	tests := []struct {
		name    string
		cluster gkeCluster
	}{
		{
			name:    "invalid cluster name",
			cluster: gkeCluster{Name: "bad/name", Location: "us-central1", Endpoint: "1.2.3.4"},
		},
		{
			name:    "invalid location",
			cluster: gkeCluster{Name: "prod", Location: "us central1", Endpoint: "1.2.3.4"},
		},
		{
			name:    "missing endpoint",
			cluster: gkeCluster{Name: "prod", Location: "us-central1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := processGKECluster(tt.cluster); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestDownloadGCPKubeConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	tokenServer := newTokenServer(t, "urn:ietf:params:oauth:grant-type:jwt-bearer")
	defer tokenServer.Close()

	gkeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"clusters": [
			{"name": "prod", "location": "us-central1", "endpoint": "1.2.3.4", "masterAuth": {"clusterCaCertificate": "Y2E="}}
		]}`)
	}))
	defer gkeServer.Close()

	origBase := gkeAPIBaseURL
	gkeAPIBaseURL = gkeServer.URL
	defer func() { gkeAPIBaseURL = origBase }()

	credsPath := writeCredsFile(t, testServiceAccountJSON(t))

	cred := credentials.Credential{
		Provider: "GCP",
		Details: map[string]string{
			"CredentialsFile": credsPath,
			"ProjectID":       "test-project",
		},
	}

	if err := downloadGCPKubeConfig(cred); err != nil {
		t.Fatalf("downloadGCPKubeConfig() error = %v", err)
	}

	kubeconfigPath := filepath.Join(tempDir, ".kube", "prod@us-central1-kubeconfig.yaml")
	data, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		t.Fatalf("expected kubeconfig at %s: %v", kubeconfigPath, err)
	}
	if !strings.Contains(string(data), "server: https://1.2.3.4") {
		t.Errorf("kubeconfig content missing server, got:\n%s", string(data))
	}
}

func TestDownloadGCPKubeConfigMissingDetails(t *testing.T) {
	cred := credentials.Credential{Provider: "GCP", Details: map[string]string{}}
	if err := downloadGCPKubeConfig(cred); err == nil {
		t.Fatal("expected error for missing credential details, got nil")
	}
}
