package kubeconfig

import (
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "strings"
    "time"
    "kubectm/pkg/utils"
)

// DefaultBackupCount is the default number of kubeconfig backups to keep.
const DefaultBackupCount = 5

// backupPrefix is the filename prefix used for kubeconfig backups in ~/.kube/.
const backupPrefix = "config.bak."

// backupTimestampFormat is the compact ISO 8601 layout used in backup
// filenames (no colons, so the name is valid on Windows too).
const backupTimestampFormat = "20060102T150405Z"

// BackupConfig copies ~/.kube/config to ~/.kube/config.bak.{timestamp} so the
// user can recover the previous state if a merge goes wrong. After creating
// the backup it prunes older backups, keeping only the most recent `keep`
// files (values below 1 are treated as 1).
//
// It returns the path of the created backup, or an empty string if there was
// no existing kubeconfig to back up.
func BackupConfig(keep int) (string, error) {
    homeDir, kubeDir, err := getKubeDir()
    if err != nil {
        return "", err
    }

    if rel, relErr := filepath.Rel(homeDir, kubeDir); relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
        return "", fmt.Errorf("invalid kubeconfig directory outside user home: %s", kubeDir)
    }

    configPath := filepath.Clean(filepath.Join(kubeDir, "config"))
    if !strings.HasPrefix(configPath, kubeDir) {
        return "", fmt.Errorf("invalid main kubeconfig path outside .kube directory: %s", configPath)
    }

    data, err := os.ReadFile(configPath)
    if err != nil {
        if os.IsNotExist(err) {
            utils.InfoLogger.Printf("%s No existing kubeconfig at %s, skipping backup", utils.Iso8601Time(), configPath)
            return "", nil
        }
        return "", fmt.Errorf("failed to read kubeconfig for backup: %v", err)
    }

    timestamp := time.Now().UTC().Format(backupTimestampFormat)
    backupPath := filepath.Clean(filepath.Join(kubeDir, backupPrefix+timestamp))
    if !strings.HasPrefix(backupPath, kubeDir) {
        return "", fmt.Errorf("invalid backup path outside .kube directory: %s", backupPath)
    }

    if err := os.WriteFile(backupPath, data, 0600); err != nil {
        return "", fmt.Errorf("failed to write kubeconfig backup: %v", err)
    }
    utils.InfoLogger.Printf("%s Backed up kubeconfig to %s", utils.Iso8601Time(), backupPath)

    if err := pruneBackups(kubeDir, keep); err != nil {
        utils.WarnLogger.Printf("%s Warning: failed to prune old kubeconfig backups: %v", utils.Iso8601Time(), err)
    }

    return backupPath, nil
}

// pruneBackups removes the oldest config.bak.* files in kubeDir, keeping the
// most recent `keep` backups. Backup filenames embed a compact ISO 8601 UTC
// timestamp, so lexical order matches chronological order.
func pruneBackups(kubeDir string, keep int) error {
    if keep < 1 {
        keep = 1
    }

    entries, err := os.ReadDir(kubeDir)
    if err != nil {
        return fmt.Errorf("failed to read kubeconfig directory: %v", err)
    }

    var backups []string
    for _, entry := range entries {
        if entry.IsDir() || !strings.HasPrefix(entry.Name(), backupPrefix) {
            continue
        }
        // Only prune files whose suffix is a timestamp we generated, so
        // manually created backups like config.bak.before-upgrade survive.
        timestampPart := strings.TrimPrefix(entry.Name(), backupPrefix)
        if _, err := time.Parse(backupTimestampFormat, timestampPart); err != nil {
            continue
        }
        backups = append(backups, entry.Name())
    }

    if len(backups) <= keep {
        return nil
    }

    sort.Strings(backups)
    for _, name := range backups[:len(backups)-keep] {
        backupPath := filepath.Clean(filepath.Join(kubeDir, name))
        if !strings.HasPrefix(backupPath, kubeDir) {
            utils.WarnLogger.Printf("%s Skipping deletion of file outside .kube directory: %s", utils.Iso8601Time(), backupPath)
            continue
        }
        if err := os.Remove(backupPath); err != nil {
            utils.WarnLogger.Printf("%s Warning: failed to delete old backup %s: %v", utils.Iso8601Time(), backupPath, err)
        } else {
            utils.InfoLogger.Printf("%s Deleted old kubeconfig backup %s", utils.Iso8601Time(), backupPath)
        }
    }

    return nil
}
