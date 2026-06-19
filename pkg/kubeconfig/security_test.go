package kubeconfig

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSaveKubeconfigToFileRejectsPathTraversal verifies that cluster labels
// derived from untrusted API responses cannot be used to write outside the
// ~/.kube directory.
func TestSaveKubeconfigToFileRejectsPathTraversal(t *testing.T) {
	tests := []struct {
		name        string
		clusterLabel string
		expectError bool
	}{
		{name: "simple label", clusterLabel: "prod-cluster", expectError: false},
		{name: "label with at sign", clusterLabel: "cluster@us-east-1", expectError: false},
		{name: "forward slash traversal", clusterLabel: "../../etc/cron.d/evil", expectError: true},
		{name: "backslash traversal", clusterLabel: `..\..\evil`, expectError: true},
		{name: "embedded slash", clusterLabel: "a/b", expectError: true},
		{name: "parent dir sequence", clusterLabel: "..", expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)

			err := saveKubeconfigToFile(tt.clusterLabel, "apiVersion: v1\nkind: Config\n")
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error for label %q, got nil", tt.clusterLabel)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for label %q: %v", tt.clusterLabel, err)
			}
			// The file must land inside ~/.kube and nowhere else.
			want := filepath.Join(home, ".kube", tt.clusterLabel+"-kubeconfig.yaml")
			if _, statErr := os.Stat(want); statErr != nil {
				t.Fatalf("expected kubeconfig at %s: %v", want, statErr)
			}
		})
	}
}

// TestIsValidEKSIdentifier verifies the allowlist used to sanitise EKS cluster
// names and regions before they are interpolated into kubeconfig YAML.
func TestIsValidEKSIdentifier(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{name: "cluster name", input: "my-eks-cluster", valid: true},
		{name: "region", input: "eu-west-2", valid: true},
		{name: "with dots and underscores", input: "cluster_1.test", valid: true},
		{name: "empty", input: "", valid: false},
		{name: "newline injection", input: "evil\n    command: rm", valid: false},
		{name: "space", input: "two words", valid: false},
		{name: "path separator", input: "../escape", valid: false},
		{name: "yaml metachar", input: "a: b", valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidEKSIdentifier(tt.input); got != tt.valid {
				t.Fatalf("isValidEKSIdentifier(%q) = %v, want %v", tt.input, got, tt.valid)
			}
		})
	}
}
