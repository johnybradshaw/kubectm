package main

import (
    "kubectm/pkg/credentials"
    "kubectm/pkg/kubeconfig"
    "kubectm/pkg/ui"
    "log"
)

func main() {
    log.Println("Starting kubectm...")

    creds, err := credentials.RetrieveAll()
    if err != nil {
        log.Fatalf("Failed to retrieve credentials: %v", err)
    }

    selectedCreds := ui.SelectCredentials(creds)

    err = kubeconfig.DownloadConfigs(selectedCreds)
    if err != nil {
        log.Fatalf("Failed to download kubeconfig files: %v", err)
    }

    err = kubeconfig.MergeConfigs()
    if err != nil {
        log.Fatalf("Failed to merge kubeconfig files: %v", err)
    }

    log.Println("kubectm finished successfully.")
}