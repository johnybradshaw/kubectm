package kubeconfig

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"kubectm/pkg/credentials"
)

func TestListProviderContextsUnsupported(t *testing.T) {
	cred := credentials.Credential{Provider: "DigitalOcean"}
	if _, err := listProviderContexts(cred); err == nil {
		t.Fatal("expected error for unsupported provider, got nil")
	}
}

func TestListGCPContexts(t *testing.T) {
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

	credsPath := writeCredsFile(t, testServiceAccountJSON(t, tokenServer.URL))
	cred := credentials.Credential{
		Provider: "GCP",
		Details: map[string]string{
			"CredentialsFile": credsPath,
			"ProjectID":       "test-project",
		},
	}

	contexts, err := listProviderContexts(cred)
	if err != nil {
		t.Fatalf("listProviderContexts() error = %v", err)
	}
	if len(contexts) != 1 || contexts[0] != "prod@us-central1" {
		t.Errorf("expected [prod@us-central1], got %v", contexts)
	}
}

func TestListAzureContexts(t *testing.T) {
	newAzureLoginServer(t)
	setAzureManagementServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"value": [{"id": %q, "name": "aks-prod", "location": "eastus"}]}`, testAKSClusterID)
	}))

	contexts, err := listProviderContexts(testAzureCredential())
	if err != nil {
		t.Fatalf("listProviderContexts() error = %v", err)
	}
	if len(contexts) != 1 || contexts[0] != "aks-prod@rg-prod" {
		t.Errorf("expected [aks-prod@rg-prod], got %v", contexts)
	}
}

// TestDryRunConfigsDoesNotModifyFiles verifies that a dry run reports
// clusters without downloading kubeconfigs or touching ~/.kube.
func TestDryRunConfigsDoesNotModifyFiles(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	// Existing config with one context that the provider also reports, so
	// both the "already exists" and "would be added" paths are exercised.
	kubeDir := filepath.Join(tempDir, ".kube")
	if err := os.MkdirAll(kubeDir, 0700); err != nil {
		t.Fatalf("failed to create .kube dir: %v", err)
	}
	existingConfig := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://1.2.3.4
  name: prod@us-central1
contexts:
- context:
    cluster: prod@us-central1
    user: prod@us-central1
  name: prod@us-central1
users:
- name: prod@us-central1
  user:
    token: t
`
	configPath := filepath.Join(kubeDir, "config")
	if err := os.WriteFile(configPath, []byte(existingConfig), 0600); err != nil {
		t.Fatalf("failed to write existing config: %v", err)
	}

	tokenServer := newTokenServer(t, "urn:ietf:params:oauth:grant-type:jwt-bearer")
	defer tokenServer.Close()

	gkeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"clusters": [
			{"name": "prod", "location": "us-central1", "endpoint": "1.2.3.4", "masterAuth": {"clusterCaCertificate": "Y2E="}},
			{"name": "new-cluster", "location": "europe-west1", "endpoint": "5.6.7.8", "masterAuth": {"clusterCaCertificate": "Y2E="}}
		]}`)
	}))
	defer gkeServer.Close()

	origBase := gkeAPIBaseURL
	gkeAPIBaseURL = gkeServer.URL
	defer func() { gkeAPIBaseURL = origBase }()

	credsPath := writeCredsFile(t, testServiceAccountJSON(t, tokenServer.URL))
	creds := []credentials.Credential{{
		Provider: "GCP",
		Details: map[string]string{
			"CredentialsFile": credsPath,
			"ProjectID":       "test-project",
		},
	}}

	if err := DryRunConfigs(creds); err != nil {
		t.Fatalf("DryRunConfigs() error = %v", err)
	}

	// The main config must be untouched and no temporary kubeconfigs written.
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config after dry run: %v", err)
	}
	if string(data) != existingConfig {
		t.Error("expected ~/.kube/config to be unchanged after dry run")
	}

	entries, err := os.ReadDir(kubeDir)
	if err != nil {
		t.Fatalf("failed to read .kube dir: %v", err)
	}
	for _, entry := range entries {
		if entry.Name() != "config" {
			t.Errorf("unexpected file created during dry run: %s", entry.Name())
		}
	}
}

// TestDryRunConfigsProviderFailure verifies that a failing provider is
// logged and skipped without aborting the dry run.
func TestDryRunConfigsProviderFailure(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	creds := []credentials.Credential{{
		Provider: "GCP",
		Details:  map[string]string{}, // missing everything
	}}

	if err := DryRunConfigs(creds); err != nil {
		t.Fatalf("DryRunConfigs() should tolerate provider failures, got error = %v", err)
	}
}
