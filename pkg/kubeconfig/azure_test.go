package kubeconfig

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kubectm/pkg/credentials"
)

const testAKSClusterID = "/subscriptions/sub-id/resourcegroups/rg-prod/providers/Microsoft.ContainerService/managedClusters/aks-prod"

// testAzureCredential returns a complete Azure service principal credential.
func testAzureCredential() credentials.Credential {
	return credentials.Credential{
		Provider: "Azure",
		Details: map[string]string{
			"ClientID":       "client-id",
			"ClientSecret":   "client-secret",
			"TenantID":       "tenant-id",
			"SubscriptionID": "sub-id",
		},
	}
}

// newAzureLoginServer returns an httptest server for the client credentials
// grant and points azureLoginBaseURL at it for the duration of the test.
func newAzureLoginServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("failed to parse token request form: %v", err)
		}
		if got := r.PostForm.Get("grant_type"); got != "client_credentials" {
			t.Errorf("expected grant_type client_credentials, got %q", got)
		}
		if got := r.PostForm.Get("client_id"); got != "client-id" {
			t.Errorf("expected client_id client-id, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token": "test-access-token", "token_type": "Bearer"}`)
	}))
	origLogin := azureLoginBaseURL
	azureLoginBaseURL = server.URL
	t.Cleanup(func() {
		azureLoginBaseURL = origLogin
		server.Close()
	})
	return server
}

// setAzureManagementServer points azureManagementBaseURL at the given handler
// for the duration of the test.
func setAzureManagementServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	origMgmt := azureManagementBaseURL
	azureManagementBaseURL = server.URL
	t.Cleanup(func() {
		azureManagementBaseURL = origMgmt
		server.Close()
	})
	return server
}

func TestGetAzureAccessToken(t *testing.T) {
	newAzureLoginServer(t)

	token, err := getAzureAccessToken(context.Background(), "tenant-id", "client-id", "client-secret")
	if err != nil {
		t.Fatalf("getAzureAccessToken() error = %v", err)
	}
	if token != "test-access-token" {
		t.Errorf("expected token test-access-token, got %q", token)
	}
}

func TestListAKSClustersPagination(t *testing.T) {
	var server *httptest.Server
	server = setAzureManagementServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("expected bearer token header, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("page") == "2" {
			fmt.Fprintf(w, `{"value": [{"id": %q, "name": "aks-two", "location": "westeurope"}]}`,
				"/subscriptions/sub-id/resourceGroups/rg-two/providers/Microsoft.ContainerService/managedClusters/aks-two")
			return
		}
		fmt.Fprintf(w, `{"value": [{"id": %q, "name": "aks-one", "location": "eastus"}], "nextLink": %q}`,
			testAKSClusterID, server.URL+"/subscriptions/sub-id/providers/Microsoft.ContainerService/managedClusters?page=2")
	}))

	clusters, err := listAKSClusters(context.Background(), "test-token", "sub-id")
	if err != nil {
		t.Fatalf("listAKSClusters() error = %v", err)
	}
	if len(clusters) != 2 {
		t.Fatalf("expected 2 clusters across pages, got %d", len(clusters))
	}
	if clusters[0].Name != "aks-one" || clusters[1].Name != "aks-two" {
		t.Errorf("unexpected cluster names: %s, %s", clusters[0].Name, clusters[1].Name)
	}
}

func TestListAKSClustersRejectsForeignNextLink(t *testing.T) {
	setAzureManagementServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"value": [], "nextLink": "https://evil.example.com/steal-token"}`)
	}))

	if _, err := listAKSClusters(context.Background(), "test-token", "sub-id"); err == nil {
		t.Fatal("expected error for pagination link outside management endpoint, got nil")
	}
}

func TestListAKSClustersErrorStatus(t *testing.T) {
	setAzureManagementServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"error": {"message": "forbidden"}}`)
	}))

	if _, err := listAKSClusters(context.Background(), "test-token", "sub-id"); err == nil {
		t.Fatal("expected error for non-200 response, got nil")
	}
}

func TestGetAKSClusterKubeconfig(t *testing.T) {
	kubeconfigContent := "apiVersion: v1\nkind: Config\n"
	setAzureManagementServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/listClusterUserCredential") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"kubeconfigs": [
			{"name": "clusterAdmin", "value": %q},
			{"name": "clusterUser", "value": %q}
		]}`, base64.StdEncoding.EncodeToString([]byte("admin")), base64.StdEncoding.EncodeToString([]byte(kubeconfigContent)))
	}))

	kubeconfig, err := getAKSClusterKubeconfig(context.Background(), "test-token", testAKSClusterID)
	if err != nil {
		t.Fatalf("getAKSClusterKubeconfig() error = %v", err)
	}
	if kubeconfig != kubeconfigContent {
		t.Errorf("expected clusterUser kubeconfig %q, got %q", kubeconfigContent, kubeconfig)
	}
}

func TestGetAKSClusterKubeconfigMissingClusterUser(t *testing.T) {
	setAzureManagementServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"kubeconfigs": []}`)
	}))

	if _, err := getAKSClusterKubeconfig(context.Background(), "test-token", testAKSClusterID); err == nil {
		t.Fatal("expected error when clusterUser kubeconfig is missing, got nil")
	}
}

func TestGetAKSClusterKubeconfigRejectsBadResourceID(t *testing.T) {
	if _, err := getAKSClusterKubeconfig(context.Background(), "test-token", "https://evil.example.com/"); err == nil {
		t.Fatal("expected error for malformed cluster resource ID, got nil")
	}
}

func TestResourceGroupFromID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{
			name: "lowercase segment",
			id:   testAKSClusterID,
			want: "rg-prod",
		},
		{
			name: "mixed case segment",
			id:   "/subscriptions/s/resourceGroups/MyGroup/providers/Microsoft.ContainerService/managedClusters/c",
			want: "MyGroup",
		},
		{
			name: "no resource group",
			id:   "/subscriptions/s/providers/Microsoft.ContainerService/managedClusters/c",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resourceGroupFromID(tt.id); got != tt.want {
				t.Errorf("resourceGroupFromID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsValidAzureIdentifier(t *testing.T) {
	valid := []string{"aks-prod", "rg_test", "MyGroup(dev)", "a.b-c"}
	for _, s := range valid {
		if !isValidAzureIdentifier(s) {
			t.Errorf("expected %q to be valid", s)
		}
	}
	invalid := []string{"", "has space", "path/traversal", `back\slash`, "colon:name"}
	for _, s := range invalid {
		if isValidAzureIdentifier(s) {
			t.Errorf("expected %q to be invalid", s)
		}
	}
}

func TestDownloadAzureKubeConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	newAzureLoginServer(t)

	kubeconfigContent := "apiVersion: v1\nkind: Config\n"
	setAzureManagementServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/listClusterUserCredential") {
			fmt.Fprintf(w, `{"kubeconfigs": [{"name": "clusterUser", "value": %q}]}`,
				base64.StdEncoding.EncodeToString([]byte(kubeconfigContent)))
			return
		}
		fmt.Fprintf(w, `{"value": [{"id": %q, "name": "aks-prod", "location": "eastus"}]}`, testAKSClusterID)
	}))

	if err := downloadAzureKubeConfig(testAzureCredential()); err != nil {
		t.Fatalf("downloadAzureKubeConfig() error = %v", err)
	}

	kubeconfigPath := filepath.Join(tempDir, ".kube", "aks-prod@rg-prod-kubeconfig.yaml")
	data, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		t.Fatalf("expected kubeconfig at %s: %v", kubeconfigPath, err)
	}
	if string(data) != kubeconfigContent {
		t.Errorf("kubeconfig content mismatch: got %q", string(data))
	}
}

func TestDownloadAzureKubeConfigIncompleteCredentials(t *testing.T) {
	cred := credentials.Credential{Provider: "Azure", Details: map[string]string{"ClientID": "only-id"}}
	if err := downloadAzureKubeConfig(cred); err == nil {
		t.Fatal("expected error for incomplete credentials, got nil")
	}
}
