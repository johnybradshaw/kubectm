package credentials

import (
	"os"
	"path/filepath"
	"testing"
)

const testSAProjectID = "test-project"

// clearGCPEnv removes GCP-related environment variables so tests control
// discovery precisely.
func clearGCPEnv(t *testing.T) {
	t.Helper()
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")
	t.Setenv("GOOGLE_CLOUD_PROJECT", "")
}

// writeGCPFile writes content to path, creating parent directories.
func writeGCPFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func TestRetrieveGCPCredentialsFromEnvVar(t *testing.T) {
	clearGCPEnv(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	saPath := filepath.Join(home, "sa.json")
	writeGCPFile(t, saPath, `{
		"type": "service_account",
		"project_id": "test-project",
		"client_email": "svc@test-project.iam.gserviceaccount.com"
	}`)
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", saPath)

	cred, err := retrieveGCPCredentials()
	if err != nil {
		t.Fatalf("retrieveGCPCredentials() error = %v", err)
	}
	if cred == nil {
		t.Fatal("expected credentials, got nil")
	}
	if cred.Provider != "GCP" {
		t.Errorf("expected provider GCP, got %s", cred.Provider)
	}
	if cred.Details["ProjectID"] != testSAProjectID {
		t.Errorf("expected project %s, got %s", testSAProjectID, cred.Details["ProjectID"])
	}
	if cred.Details["CredentialsFile"] != saPath {
		t.Errorf("expected credentials file %s, got %s", saPath, cred.Details["CredentialsFile"])
	}
	if cred.Details["ClientEmail"] != "svc@test-project.iam.gserviceaccount.com" {
		t.Errorf("unexpected client email %s", cred.Details["ClientEmail"])
	}
}

func TestRetrieveGCPCredentialsEnvVarMissingFile(t *testing.T) {
	clearGCPEnv(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/nonexistent/sa.json")

	if _, err := retrieveGCPCredentials(); err == nil {
		t.Fatal("expected error for unreadable GOOGLE_APPLICATION_CREDENTIALS, got nil")
	}
}

func TestRetrieveGCPCredentialsFromADC(t *testing.T) {
	clearGCPEnv(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	adcPath := filepath.Join(home, ".config", "gcloud", "application_default_credentials.json")
	writeGCPFile(t, adcPath, `{
		"type": "authorized_user",
		"client_id": "id",
		"client_secret": "secret",
		"refresh_token": "token"
	}`)
	// Project comes from the gcloud active configuration.
	writeGCPFile(t, filepath.Join(home, ".config", "gcloud", "configurations", "config_default"), `[core]
account = user@example.com
project = gcloud-project
`)

	cred, err := retrieveGCPCredentials()
	if err != nil {
		t.Fatalf("retrieveGCPCredentials() error = %v", err)
	}
	if cred == nil {
		t.Fatal("expected credentials, got nil")
	}
	if cred.Details["CredentialsFile"] != adcPath {
		t.Errorf("expected credentials file %s, got %s", adcPath, cred.Details["CredentialsFile"])
	}
	if cred.Details["ProjectID"] != "gcloud-project" {
		t.Errorf("expected project gcloud-project, got %s", cred.Details["ProjectID"])
	}
}

func TestRetrieveGCPCredentialsProjectFromEnv(t *testing.T) {
	clearGCPEnv(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	adcPath := filepath.Join(home, ".config", "gcloud", "application_default_credentials.json")
	writeGCPFile(t, adcPath, `{"type": "authorized_user"}`)
	t.Setenv("GOOGLE_CLOUD_PROJECT", "env-project")

	cred, err := retrieveGCPCredentials()
	if err != nil {
		t.Fatalf("retrieveGCPCredentials() error = %v", err)
	}
	if cred == nil {
		t.Fatal("expected credentials, got nil")
	}
	if cred.Details["ProjectID"] != "env-project" {
		t.Errorf("expected project env-project, got %s", cred.Details["ProjectID"])
	}
}

func TestRetrieveGCPCredentialsNoProject(t *testing.T) {
	clearGCPEnv(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	adcPath := filepath.Join(home, ".config", "gcloud", "application_default_credentials.json")
	writeGCPFile(t, adcPath, `{"type": "authorized_user"}`)

	cred, err := retrieveGCPCredentials()
	if err != nil {
		t.Fatalf("retrieveGCPCredentials() error = %v", err)
	}
	if cred != nil {
		t.Errorf("expected nil credential when no project is configured, got %+v", cred)
	}
}

func TestRetrieveGCPCredentialsNotFound(t *testing.T) {
	clearGCPEnv(t)
	t.Setenv("HOME", t.TempDir())

	cred, err := retrieveGCPCredentials()
	if err != nil {
		t.Fatalf("retrieveGCPCredentials() error = %v", err)
	}
	if cred != nil {
		t.Errorf("expected nil credential, got %+v", cred)
	}
}

func TestParseGcloudProject(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "project in core section",
			content: "[core]\nproject = my-project\n",
			want:    "my-project",
		},
		{
			name:    "project outside core section ignored",
			content: "[compute]\nproject = wrong\n[core]\naccount = a@b.c\n",
			want:    "",
		},
		{
			name:    "comments and blank lines skipped",
			content: "# comment\n\n[core]\n; another\nproject=tight-spacing\n",
			want:    "tight-spacing",
		},
		{
			name:    "empty file",
			content: "",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseGcloudProject([]byte(tt.content)); got != tt.want {
				t.Errorf("parseGcloudProject() = %q, want %q", got, tt.want)
			}
		})
	}
}
