package kubeconfig

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"kubectm/pkg/credentials"
	"kubectm/pkg/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awscreds "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/fatih/color"
)

const (
	awsDownloadTimeout = 30 * time.Second
	awsConcurrencyLimit = 5
)

// kubectmConfig represents the optional ~/.kubectm/config.json file.
type kubectmConfig struct {
	AWSRegions []string `json:"aws_regions"`
}

// downloadAWSKubeConfig downloads EKS cluster kubeconfigs for all enabled regions.
// It uses EC2 DescribeRegions to auto-discover regions, with an optional override
// via ~/.kubectm/config.json. Regions are scanned in parallel with bounded concurrency.
func downloadAWSKubeConfig(cred credentials.Credential) error {
	ctx, cancel := context.WithTimeout(context.Background(), awsDownloadTimeout)
	defer cancel()

	cfg, err := newAWSConfig(ctx, cred)
	if err != nil {
		return fmt.Errorf("failed to create AWS config: %v", err)
	}

	regions, err := getAWSRegions(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to get AWS regions: %v", err)
	}

	if len(regions) == 0 {
		utils.WarnLogger.Printf("%s No AWS regions found to scan for EKS clusters", utils.Iso8601Time())
		return nil
	}

	utils.InfoLogger.Printf("%s Scanning %d AWS regions for EKS clusters", utils.Iso8601Time(), len(regions))

	return scanRegionsForClusters(ctx, cfg, regions)
}

// newAWSConfig creates an AWS SDK config from the discovered credentials.
func newAWSConfig(ctx context.Context, cred credentials.Credential) (aws.Config, error) {
	accessKey := cred.Details["AccessKey"]
	secretKey := cred.Details["SecretKey"]
	if accessKey == "" || secretKey == "" {
		return aws.Config{}, fmt.Errorf("AWS access key or secret key is missing")
	}

	staticCreds := awscreds.NewStaticCredentialsProvider(
		accessKey, secretKey, cred.Details["SessionToken"],
	)

	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithCredentialsProvider(staticCreds),
	}

	if region := cred.Details["Region"]; region != "" {
		opts = append(opts, awsconfig.WithRegion(region))
	} else {
		opts = append(opts, awsconfig.WithRegion("us-east-1"))
	}

	return awsconfig.LoadDefaultConfig(ctx, opts...)
}

// getAWSRegions returns the list of AWS regions to scan. It checks for a config
// override in ~/.kubectm/config.json first, then falls back to EC2 DescribeRegions.
func getAWSRegions(ctx context.Context, cfg aws.Config) ([]string, error) {
	regions, err := loadRegionOverride()
	if err != nil {
		utils.WarnLogger.Printf("%s Error reading config override, falling back to auto-discover: %v", utils.Iso8601Time(), err)
	}
	if len(regions) > 0 {
		utils.InfoLogger.Printf("%s Using configured AWS regions: %v", utils.Iso8601Time(), regions)
		return regions, nil
	}

	return discoverEnabledRegions(ctx, cfg)
}

// loadRegionOverride reads the optional ~/.kubectm/config.json for an aws_regions override.
func loadRegionOverride() ([]string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	homeDir = filepath.Clean(homeDir)

	configPath := filepath.Clean(filepath.Join(homeDir, ".kubectm", "config.json"))
	if !strings.HasPrefix(configPath, homeDir+string(filepath.Separator)) {
		return nil, fmt.Errorf("invalid config path outside user home")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	var config kubectmConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}

	return config.AWSRegions, nil
}

// discoverEnabledRegions calls EC2 DescribeRegions to get all enabled regions.
func discoverEnabledRegions(ctx context.Context, cfg aws.Config) ([]string, error) {
	client := ec2.NewFromConfig(cfg)
	output, err := client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("opt-in-status"),
				Values: []string{"opt-in-not-required", "opted-in"},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("EC2 DescribeRegions failed: %v", err)
	}

	regions := make([]string, 0, len(output.Regions))
	for _, r := range output.Regions {
		if r.RegionName != nil {
			regions = append(regions, *r.RegionName)
		}
	}

	return regions, nil
}

// scanRegionsForClusters scans all given regions for EKS clusters in parallel
// with bounded concurrency. Per-region errors are logged and skipped.
func scanRegionsForClusters(ctx context.Context, cfg aws.Config, regions []string) error {
	sem := make(chan struct{}, awsConcurrencyLimit)
	var mu sync.Mutex
	var errs []string

	var wg sync.WaitGroup
	for _, region := range regions {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := processRegion(ctx, cfg, region); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Sprintf("%s: %v", region, err))
				mu.Unlock()
				utils.WarnLogger.Printf("%s Failed to scan region %s: %v", utils.Iso8601Time(), region, err)
			}
		}(region)
	}
	wg.Wait()

	if len(errs) == len(regions) {
		return fmt.Errorf("all regions failed: %s", strings.Join(errs, "; "))
	}

	if len(errs) > 0 {
		utils.WarnLogger.Printf("%s %d/%d regions had errors", utils.Iso8601Time(), len(errs), len(regions))
	}

	return nil
}

// processRegion lists EKS clusters in a single region and downloads kubeconfigs for each.
func processRegion(ctx context.Context, cfg aws.Config, region string) error {
	regionalCfg := cfg.Copy()
	regionalCfg.Region = region

	eksClient := eks.NewFromConfig(regionalCfg)

	clusters, err := listEKSClusters(ctx, eksClient)
	if err != nil {
		return fmt.Errorf("failed to list clusters: %v", err)
	}

	if len(clusters) == 0 {
		return nil
	}

	utils.InfoLogger.Printf("%s Found %d EKS cluster(s) in %s", utils.Iso8601Time(), len(clusters), region)

	for _, clusterName := range clusters {
		if err := processEKSCluster(ctx, eksClient, clusterName, region); err != nil {
			utils.WarnLogger.Printf("%s Failed to process cluster %s in %s: %v", utils.Iso8601Time(), clusterName, region, err)
			continue
		}
	}

	return nil
}

// listEKSClusters lists all EKS cluster names in the region, handling pagination.
func listEKSClusters(ctx context.Context, client *eks.Client) ([]string, error) {
	var allClusters []string
	var nextToken *string

	for {
		output, err := client.ListClusters(ctx, &eks.ListClustersInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}

		allClusters = append(allClusters, output.Clusters...)

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return allClusters, nil
}

// processEKSCluster describes a single EKS cluster and generates + saves a kubeconfig for it.
func processEKSCluster(ctx context.Context, client *eks.Client, clusterName, region string) error {
	output, err := client.DescribeCluster(ctx, &eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	})
	if err != nil {
		return fmt.Errorf("DescribeCluster failed: %v", err)
	}

	cluster := output.Cluster
	if cluster == nil || cluster.Endpoint == nil || cluster.CertificateAuthority == nil || cluster.CertificateAuthority.Data == nil {
		return fmt.Errorf("cluster %s has incomplete data", clusterName)
	}

	contextName := fmt.Sprintf("%s@%s", clusterName, region)
	utils.ActionLogger.Printf("%s Downloading kubeconfig for EKS cluster: %s",
		utils.Iso8601Time(), color.New(color.Bold).Sprint(contextName))

	kubeconfigContent := generateEKSKubeconfig(
		clusterName,
		region,
		*cluster.Endpoint,
		*cluster.CertificateAuthority.Data,
	)

	return saveKubeconfigToFile(contextName, kubeconfigContent)
}

// eksKubeconfigTemplate is the template for generating EKS kubeconfig files.
var eksKubeconfigTemplate = template.Must(template.New("kubeconfig").Parse(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: {{.Endpoint}}
    certificate-authority-data: {{.CAData}}
  name: {{.ContextName}}
contexts:
- context:
    cluster: {{.ContextName}}
    user: {{.ContextName}}
  name: {{.ContextName}}
current-context: {{.ContextName}}
users:
- name: {{.ContextName}}
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      command: aws
      args:
      - eks
      - get-token
      - --cluster-name
      - {{.ClusterName}}
      - --region
      - {{.Region}}
`))

// generateEKSKubeconfig generates a kubeconfig YAML string for an EKS cluster
// that uses the `aws eks get-token` exec plugin for authentication.
func generateEKSKubeconfig(clusterName, region, endpoint, caData string) string {
	data := struct {
		Endpoint    string
		CAData      string
		ContextName string
		ClusterName string
		Region      string
	}{
		Endpoint:    endpoint,
		CAData:      caData,
		ContextName: fmt.Sprintf("%s@%s", clusterName, region),
		ClusterName: clusterName,
		Region:      region,
	}

	var buf bytes.Buffer
	if err := eksKubeconfigTemplate.Execute(&buf, data); err != nil {
		utils.ErrorLogger.Printf("%s Error executing EKS kubeconfig template: %v", utils.Iso8601Time(), err)
		return ""
	}

	return buf.String()
}
