package credentials

import (
	"os"
	"path/filepath"
	"testing"
)

// clearAzureEnv removes Azure-related environment variables so tests control
// discovery precisely.
func clearAzureEnv(t *testing.T) {
	t.Helper()
	t.Setenv("AZURE_CLIENT_ID", "")
	t.Setenv("AZURE_CLIENT_SECRET", "")
	t.Setenv("AZURE_TENANT_ID", "")
	t.Setenv("AZURE_SUBSCRIPTION_ID", "")
}

func setAzureServicePrincipalEnv(t *testing.T) {
	t.Helper()
	t.Setenv("AZURE_CLIENT_ID", "client-id")
	t.Setenv("AZURE_CLIENT_SECRET", "client-secret")
	t.Setenv("AZURE_TENANT_ID", "tenant-id")
}

func TestRetrieveAzureCredentialsFromEnvVars(t *testing.T) {
	clearAzureEnv(t)
	t.Setenv("HOME", t.TempDir())
	setAzureServicePrincipalEnv(t)
	t.Setenv("AZURE_SUBSCRIPTION_ID", "sub-id")

	cred, err := retrieveAzureCredentials()
	if err != nil {
		t.Fatalf("retrieveAzureCredentials() error = %v", err)
	}
	if cred == nil {
		t.Fatal("expected credentials, got nil")
	}
	if cred.Provider != "Azure" {
		t.Errorf("expected provider Azure, got %s", cred.Provider)
	}
	for key, want := range map[string]string{
		"ClientID":       "client-id",
		"ClientSecret":   "client-secret",
		"TenantID":       "tenant-id",
		"SubscriptionID": "sub-id",
	} {
		if got := cred.Details[key]; got != want {
			t.Errorf("expected %s=%s, got %s", key, want, got)
		}
	}
}

func TestRetrieveAzureCredentialsIncompletePrincipal(t *testing.T) {
	clearAzureEnv(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AZURE_CLIENT_ID", "client-id")
	// No client secret or tenant.

	cred, err := retrieveAzureCredentials()
	if err != nil {
		t.Fatalf("retrieveAzureCredentials() error = %v", err)
	}
	if cred != nil {
		t.Errorf("expected nil credential for incomplete service principal, got %+v", cred)
	}
}

func TestRetrieveAzureCredentialsSubscriptionFromCLIProfile(t *testing.T) {
	clearAzureEnv(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	setAzureServicePrincipalEnv(t)

	profilePath := filepath.Join(home, ".azure", "azureProfile.json")
	if err := os.MkdirAll(filepath.Dir(profilePath), 0700); err != nil {
		t.Fatalf("failed to create .azure dir: %v", err)
	}
	// The Azure CLI writes this file with a UTF-8 BOM; include one to verify
	// it is handled.
	profile := "\ufeff" + `{
		"subscriptions": [
			{"id": "other-sub", "tenantId": "tenant-id", "isDefault": false},
			{"id": "default-sub", "tenantId": "tenant-id", "isDefault": true}
		]
	}`
	if err := os.WriteFile(profilePath, []byte(profile), 0600); err != nil {
		t.Fatalf("failed to write profile: %v", err)
	}

	cred, err := retrieveAzureCredentials()
	if err != nil {
		t.Fatalf("retrieveAzureCredentials() error = %v", err)
	}
	if cred == nil {
		t.Fatal("expected credentials, got nil")
	}
	if cred.Details["SubscriptionID"] != "default-sub" {
		t.Errorf("expected subscription default-sub, got %s", cred.Details["SubscriptionID"])
	}
}

func TestRetrieveAzureCredentialsNoSubscription(t *testing.T) {
	clearAzureEnv(t)
	t.Setenv("HOME", t.TempDir())
	setAzureServicePrincipalEnv(t)

	cred, err := retrieveAzureCredentials()
	if err != nil {
		t.Fatalf("retrieveAzureCredentials() error = %v", err)
	}
	if cred != nil {
		t.Errorf("expected nil credential when no subscription is configured, got %+v", cred)
	}
}

func TestRetrieveAzureCredentialsNotFound(t *testing.T) {
	clearAzureEnv(t)
	t.Setenv("HOME", t.TempDir())

	cred, err := retrieveAzureCredentials()
	if err != nil {
		t.Fatalf("retrieveAzureCredentials() error = %v", err)
	}
	if cred != nil {
		t.Errorf("expected nil credential, got %+v", cred)
	}
}
