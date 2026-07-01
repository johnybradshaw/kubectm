package kubeconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testKubeconfigContent = `apiVersion: v1
kind: Config
clusters: []
contexts: []
users: []
`

// setupBackupTestHome creates a temp home with a ~/.kube directory and
// returns the .kube dir path.
func setupBackupTestHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	kubeDir := filepath.Join(home, ".kube")
	if err := os.MkdirAll(kubeDir, 0700); err != nil {
		t.Fatalf("failed to create .kube dir: %v", err)
	}
	return kubeDir
}

// writeTestConfig writes a kubeconfig file at ~/.kube/config.
func writeTestConfig(t *testing.T, kubeDir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(kubeDir, "config"), []byte(testKubeconfigContent), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}
}

// listBackups returns the names of config.bak.* files in kubeDir, sorted by ReadDir.
func listBackups(t *testing.T, kubeDir string) []string {
	t.Helper()
	entries, err := os.ReadDir(kubeDir)
	if err != nil {
		t.Fatalf("failed to read kube dir: %v", err)
	}
	var backups []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), backupPrefix) {
			backups = append(backups, entry.Name())
		}
	}
	return backups
}

// TestBackupConfigNoExistingConfig verifies that no backup is created and no
// error is returned when there is no kubeconfig to back up.
func TestBackupConfigNoExistingConfig(t *testing.T) {
	kubeDir := setupBackupTestHome(t)

	backupPath, err := BackupConfig(DefaultBackupCount)
	if err != nil {
		t.Fatalf("BackupConfig() error = %v", err)
	}
	if backupPath != "" {
		t.Errorf("expected empty backup path, got %q", backupPath)
	}
	if backups := listBackups(t, kubeDir); len(backups) != 0 {
		t.Errorf("expected no backup files, got %v", backups)
	}
}

// TestBackupConfigCreatesBackup verifies that the existing kubeconfig is
// copied to a timestamped backup file with matching content.
func TestBackupConfigCreatesBackup(t *testing.T) {
	kubeDir := setupBackupTestHome(t)
	writeTestConfig(t, kubeDir)

	backupPath, err := BackupConfig(DefaultBackupCount)
	if err != nil {
		t.Fatalf("BackupConfig() error = %v", err)
	}
	if backupPath == "" {
		t.Fatal("expected a backup path, got empty string")
	}
	if !strings.HasPrefix(filepath.Base(backupPath), backupPrefix) {
		t.Errorf("expected backup filename to start with %q, got %q", backupPrefix, filepath.Base(backupPath))
	}

	data, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("failed to read backup file: %v", err)
	}
	if string(data) != testKubeconfigContent {
		t.Errorf("backup content mismatch: got %q, want %q", string(data), testKubeconfigContent)
	}

	info, err := os.Stat(backupPath)
	if err != nil {
		t.Fatalf("failed to stat backup file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected backup permissions 0600, got %v", info.Mode().Perm())
	}
}

// TestBackupConfigPrunesOldBackups verifies that only the most recent `keep`
// backups survive and that the oldest are deleted.
func TestBackupConfigPrunesOldBackups(t *testing.T) {
	kubeDir := setupBackupTestHome(t)
	writeTestConfig(t, kubeDir)

	// Pre-existing backups with timestamps guaranteed older than the one
	// BackupConfig is about to create.
	oldBackups := []string{
		"config.bak.20200101T000000Z",
		"config.bak.20200102T000000Z",
		"config.bak.20200103T000000Z",
		"config.bak.20200104T000000Z",
		"config.bak.20200105T000000Z",
	}
	for _, name := range oldBackups {
		if err := os.WriteFile(filepath.Join(kubeDir, name), []byte("old"), 0600); err != nil {
			t.Fatalf("failed to create old backup %s: %v", name, err)
		}
	}

	if _, err := BackupConfig(3); err != nil {
		t.Fatalf("BackupConfig() error = %v", err)
	}

	backups := listBackups(t, kubeDir)
	if len(backups) != 3 {
		t.Fatalf("expected 3 backups after pruning, got %d: %v", len(backups), backups)
	}
	for _, name := range backups {
		if name == "config.bak.20200101T000000Z" || name == "config.bak.20200102T000000Z" || name == "config.bak.20200103T000000Z" {
			t.Errorf("expected oldest backup %s to be pruned", name)
		}
	}
}

// TestBackupConfigKeepClampedToOne verifies that keep values below 1 still
// retain the newly created backup.
func TestBackupConfigKeepClampedToOne(t *testing.T) {
	kubeDir := setupBackupTestHome(t)
	writeTestConfig(t, kubeDir)

	if err := os.WriteFile(filepath.Join(kubeDir, "config.bak.20200101T000000Z"), []byte("old"), 0600); err != nil {
		t.Fatalf("failed to create old backup: %v", err)
	}

	backupPath, err := BackupConfig(0)
	if err != nil {
		t.Fatalf("BackupConfig() error = %v", err)
	}

	backups := listBackups(t, kubeDir)
	if len(backups) != 1 {
		t.Fatalf("expected exactly 1 backup, got %d: %v", len(backups), backups)
	}
	if backups[0] != filepath.Base(backupPath) {
		t.Errorf("expected surviving backup to be %s, got %s", filepath.Base(backupPath), backups[0])
	}
}

// TestBackupConfigLeavesOtherFilesAlone verifies pruning only touches
// config.bak.* files, not the main config or temporary kubeconfig files.
func TestBackupConfigLeavesOtherFilesAlone(t *testing.T) {
	kubeDir := setupBackupTestHome(t)
	writeTestConfig(t, kubeDir)

	otherFile := filepath.Join(kubeDir, "my-cluster-kubeconfig.yaml")
	if err := os.WriteFile(otherFile, []byte("temp kubeconfig"), 0600); err != nil {
		t.Fatalf("failed to create temp kubeconfig: %v", err)
	}

	if _, err := BackupConfig(1); err != nil {
		t.Fatalf("BackupConfig() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(kubeDir, "config")); err != nil {
		t.Errorf("expected main config to be untouched: %v", err)
	}
	if _, err := os.Stat(otherFile); err != nil {
		t.Errorf("expected temporary kubeconfig to be untouched: %v", err)
	}
}
