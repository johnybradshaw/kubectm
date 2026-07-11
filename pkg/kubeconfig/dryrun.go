package kubeconfig

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"kubectm/pkg/credentials"
	"kubectm/pkg/utils"

	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/fatih/color"
	"k8s.io/client-go/tools/clientcmd/api"
)

// DryRunConfigs lists the clusters available from each provider and reports
// what a real run would change in ~/.kube/config, without downloading
// kubeconfigs or modifying any files. Per-provider errors are logged and
// skipped so one failing provider does not hide the others.
func DryRunConfigs(creds []credentials.Credential) error {
	existing, err := loadExistingContextNames()
	if err != nil {
		return err
	}

	var wouldAdd, alreadyPresent int
	for _, cred := range creds {
		contexts, err := listProviderContexts(cred)
		if err != nil {
			utils.WarnLogger.Printf("%s Dry-run: failed to list %s clusters: %v", utils.Iso8601Time(), cred.Provider, err)
			continue
		}

		if len(contexts) == 0 {
			utils.InfoLogger.Printf("%s Dry-run: no %s clusters found", utils.Iso8601Time(), cred.Provider)
			continue
		}

		sort.Strings(contexts)
		for _, contextName := range contexts {
			if existing[contextName] {
				alreadyPresent++
				utils.InfoLogger.Printf("%s Dry-run: context %s already exists (would refresh)", utils.Iso8601Time(), color.New(color.Bold).Sprint(contextName))
			} else {
				wouldAdd++
				utils.ActionLogger.Printf("%s Dry-run: context %s would be added", utils.Iso8601Time(), color.New(color.Bold).Sprint(contextName))
			}
		}
	}

	utils.InfoLogger.Printf("%s Dry-run complete: %d context(s) would be added, %d already present. No files were modified.",
		utils.Iso8601Time(), wouldAdd, alreadyPresent)
	return nil
}

// loadExistingContextNames returns the set of context names in the current
// ~/.kube/config, or an empty set if no config exists yet.
func loadExistingContextNames() (map[string]bool, error) {
	_, kubeDir, err := getKubeDir()
	if err != nil {
		return nil, err
	}

	mainKubeconfigPath := filepath.Clean(filepath.Join(kubeDir, "config"))
	config, err := loadKubeconfig(mainKubeconfigPath)
	if err != nil {
		// A missing or unreadable config just means every context is new.
		config = api.NewConfig()
	}

	existing := make(map[string]bool, len(config.Contexts))
	for name := range config.Contexts {
		existing[name] = true
	}
	return existing, nil
}

// listProviderContexts lists the context names a real run would create for
// the given provider credential, without writing anything.
func listProviderContexts(cred credentials.Credential) ([]string, error) {
	switch cred.Provider {
	case "Linode":
		return listLinodeContexts(cred)
	case "AWS":
		return listAWSClusterContexts(cred)
	case "GCP":
		return listGCPContexts(cred)
	case "Azure":
		return listAzureContexts(cred)
	default:
		return nil, fmt.Errorf("provider %s is not supported", cred.Provider)
	}
}

// listLinodeContexts lists LKE cluster labels; the merge names each context
// after the cluster label.
func listLinodeContexts(cred credentials.Credential) ([]string, error) {
	token := cred.Details["AccessToken"]
	if token == "" {
		return nil, fmt.Errorf("Linode access token is missing")
	}

	clusters, err := getLinodeClusters(token)
	if err != nil {
		return nil, err
	}

	contexts := make([]string, 0, len(clusters))
	for _, cluster := range clusters {
		contexts = append(contexts, cluster.Label)
	}
	return contexts, nil
}

// listAWSClusterContexts lists EKS cluster contexts ({name}@{region}) across
// all regions, scanning in parallel with bounded concurrency like the real
// download. Per-region errors are logged and skipped.
func listAWSClusterContexts(cred credentials.Credential) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), awsDownloadTimeout)
	defer cancel()

	cfg, err := newAWSConfig(ctx, cred)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS config: %v", err)
	}

	regions, err := getAWSRegions(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS regions: %v", err)
	}

	sem := make(chan struct{}, awsConcurrencyLimit)
	var mu sync.Mutex
	var contexts []string
	var errs []string

	var wg sync.WaitGroup
	for _, region := range regions {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			regionalCfg := cfg.Copy()
			regionalCfg.Region = region
			clusters, err := listEKSClusters(ctx, eks.NewFromConfig(regionalCfg))
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", region, err))
				return
			}
			for _, name := range clusters {
				contexts = append(contexts, fmt.Sprintf("%s@%s", name, region))
			}
		}(region)
	}
	wg.Wait()

	if len(regions) > 0 && len(errs) == len(regions) {
		return nil, fmt.Errorf("all regions failed: %s", strings.Join(errs, "; "))
	}
	if len(errs) > 0 {
		utils.WarnLogger.Printf("%s Dry-run: %d/%d AWS regions had errors", utils.Iso8601Time(), len(errs), len(regions))
	}

	return contexts, nil
}

// listGCPContexts lists GKE cluster contexts ({name}@{location}).
func listGCPContexts(cred credentials.Credential) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gcpDownloadTimeout)
	defer cancel()

	clusters, err := listGCPClusters(ctx, cred)
	if err != nil {
		return nil, err
	}

	contexts := make([]string, 0, len(clusters))
	for _, cluster := range clusters {
		contexts = append(contexts, fmt.Sprintf("%s@%s", cluster.Name, cluster.Location))
	}
	return contexts, nil
}

// listAzureContexts lists AKS cluster contexts ({name}@{resource-group}).
func listAzureContexts(cred credentials.Credential) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), azureDownloadTimeout)
	defer cancel()

	_, clusters, err := listAzureClustersWithToken(ctx, cred)
	if err != nil {
		return nil, err
	}

	contexts := make([]string, 0, len(clusters))
	for _, cluster := range clusters {
		contexts = append(contexts, fmt.Sprintf("%s@%s", cluster.Name, resourceGroupFromID(cluster.ID)))
	}
	return contexts, nil
}
