package kubeconfig

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"kubectm/pkg/credentials"
)

func TestLoadRegionOverride(t *testing.T) {
	tests := []struct {
		name           string
		configContent  string
		createFile     bool
		expectedRegions []string
		expectError    bool
	}{
		{
			name:           "config file with regions",
			configContent:  `{"aws_regions": ["us-east-1", "eu-west-1"]}`,
			createFile:     true,
			expectedRegions: []string{"us-east-1", "eu-west-1"},
			expectError:    false,
		},
		{
			name:           "config file with empty regions",
			configContent:  `{"aws_regions": []}`,
			createFile:     true,
			expectedRegions: nil,
			expectError:    false,
		},
		{
			name:           "config file does not exist",
			createFile:     false,
			expectedRegions: nil,
			expectError:    false,
		},
		{
			name:           "malformed JSON",
			configContent:  `{invalid json`,
			createFile:     true,
			expectedRegions: nil,
			expectError:    true,
		},
		{
			name:           "config file without aws_regions key",
			configContent:  `{"other_key": "value"}`,
			createFile:     true,
			expectedRegions: nil,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			origHome := os.Getenv("HOME")
			os.Setenv("HOME", tempDir)
			defer os.Setenv("HOME", origHome)

			if tt.createFile {
				configDir := filepath.Join(tempDir, ".kubectm")
				if err := os.MkdirAll(configDir, 0700); err != nil {
					t.Fatalf("failed to create config dir: %v", err)
				}
				configPath := filepath.Join(configDir, "config.json")
				if err := os.WriteFile(configPath, []byte(tt.configContent), 0600); err != nil {
					t.Fatalf("failed to write config file: %v", err)
				}
			}

			regions, err := loadRegionOverride()

			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(tt.expectedRegions) == 0 {
				if len(regions) != 0 {
					t.Errorf("expected no regions, got %v", regions)
				}
				return
			}

			if len(regions) != len(tt.expectedRegions) {
				t.Errorf("expected %d regions, got %d", len(tt.expectedRegions), len(regions))
				return
			}

			for i, r := range regions {
				if r != tt.expectedRegions[i] {
					t.Errorf("region[%d]: expected %s, got %s", i, tt.expectedRegions[i], r)
				}
			}
		})
	}
}

func TestGenerateEKSKubeconfig(t *testing.T) {
	tests := []struct {
		name        string
		clusterName string
		region      string
		endpoint    string
		caData      string
		checks      []string
	}{
		{
			name:        "standard cluster",
			clusterName: "prod-cluster",
			region:      "us-east-1",
			endpoint:    "https://ABCDEF.gr7.us-east-1.eks.amazonaws.com",
			caData:      "LS0tLS1CRUdJTi...",
			checks: []string{
				"server: https://ABCDEF.gr7.us-east-1.eks.amazonaws.com",
				"certificate-authority-data: LS0tLS1CRUdJTi...",
				"name: prod-cluster@us-east-1",
				"current-context: prod-cluster@us-east-1",
				"command: aws",
				"- get-token",
				"- --cluster-name",
				"- prod-cluster",
				"- --region",
				"- us-east-1",
				"apiVersion: client.authentication.k8s.io/v1beta1",
			},
		},
		{
			name:        "cluster with special characters",
			clusterName: "my-test-cluster-123",
			region:      "eu-west-1",
			endpoint:    "https://XYZ.gr7.eu-west-1.eks.amazonaws.com",
			caData:      "dGVzdC1jYS1kYXRh",
			checks: []string{
				"name: my-test-cluster-123@eu-west-1",
				"- my-test-cluster-123",
				"- eu-west-1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateEKSKubeconfig(tt.clusterName, tt.region, tt.endpoint, tt.caData)

			for _, check := range tt.checks {
				if !strings.Contains(result, check) {
					t.Errorf("generated kubeconfig missing expected content: %q\nGot:\n%s", check, result)
				}
			}
		})
	}
}

func TestDownloadAWSKubeConfig_MissingCredentials(t *testing.T) {
	tests := []struct {
		name       string
		cred       credentials.Credential
		expectErr  string
	}{
		{
			name: "missing access key",
			cred: credentials.Credential{
				Provider: "AWS",
				Details:  map[string]string{"SecretKey": "secret"},
			},
			expectErr: "access key or secret key is missing",
		},
		{
			name: "missing secret key",
			cred: credentials.Credential{
				Provider: "AWS",
				Details:  map[string]string{"AccessKey": "access"},
			},
			expectErr: "access key or secret key is missing",
		},
		{
			name: "empty details",
			cred: credentials.Credential{
				Provider: "AWS",
				Details:  map[string]string{},
			},
			expectErr: "access key or secret key is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := downloadAWSKubeConfig(tt.cred)
			if err == nil {
				t.Error("expected error, got nil")
				return
			}
			if !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("expected error containing %q, got: %v", tt.expectErr, err)
			}
		})
	}
}

func TestListEKSClusters(t *testing.T) {
	tests := []struct {
		name        string
		responses   []map[string]interface{}
		expectCount int
		expectError bool
	}{
		{
			name: "single page with clusters",
			responses: []map[string]interface{}{
				{"clusters": []string{"cluster-a", "cluster-b", "cluster-c"}},
			},
			expectCount: 3,
			expectError: false,
		},
		{
			name: "no clusters",
			responses: []map[string]interface{}{
				{"clusters": []string{}},
			},
			expectCount: 0,
			expectError: false,
		},
		{
			name: "paginated results",
			responses: []map[string]interface{}{
				{"clusters": []string{"cluster-1", "cluster-2"}, "nextToken": "token1"},
				{"clusters": []string{"cluster-3"}},
			},
			expectCount: 3,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if callCount >= len(tt.responses) {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/x-amz-json-1.1")
				json.NewEncoder(w).Encode(tt.responses[callCount])
				callCount++
			}))
			defer server.Close()

			client := eks.New(eks.Options{
				Region:       "us-east-1",
				BaseEndpoint: aws.String(server.URL),
				Credentials: aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
					return aws.Credentials{
						AccessKeyID: "test", SecretAccessKey: "test",
					}, nil
				}),
			})

			clusters, err := listEKSClusters(context.Background(), client)

			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if len(clusters) != tt.expectCount {
				t.Errorf("expected %d clusters, got %d: %v", tt.expectCount, len(clusters), clusters)
			}
		})
	}
}

func TestProcessEKSCluster(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	t.Run("successful cluster description", func(t *testing.T) {
		// AWS API JSON format for DescribeCluster
		describeResp := map[string]interface{}{
			"cluster": map[string]interface{}{
				"name":     "test-cluster",
				"endpoint": "https://test.eks.amazonaws.com",
				"certificateAuthority": map[string]interface{}{
					"data": "dGVzdC1jYS1kYXRh",
				},
				"status": "ACTIVE",
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/x-amz-json-1.1")
			json.NewEncoder(w).Encode(describeResp)
		}))
		defer server.Close()

		client := eks.New(eks.Options{
			Region:       "us-east-1",
			BaseEndpoint: aws.String(server.URL),
			Credentials: aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID: "test", SecretAccessKey: "test",
				}, nil
			}),
		})

		err := processEKSCluster(context.Background(), client, "test-cluster", "us-east-1")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		expectedFile := filepath.Join(tempDir, ".kube", "test-cluster@us-east-1-kubeconfig.yaml")
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			t.Errorf("expected kubeconfig file at %s", expectedFile)
			return
		}

		content, err := os.ReadFile(expectedFile)
		if err != nil {
			t.Fatalf("failed to read kubeconfig: %v", err)
		}

		checks := []string{
			"server: https://test.eks.amazonaws.com",
			"certificate-authority-data: dGVzdC1jYS1kYXRh",
			"name: test-cluster@us-east-1",
			"command: aws",
			"- get-token",
		}
		for _, check := range checks {
			if !strings.Contains(string(content), check) {
				t.Errorf("kubeconfig missing: %q", check)
			}
		}
	})
}

func TestScanRegionsForClusters_AllFail(t *testing.T) {
	// Create a server that always returns an error for EKS ListClusters
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"Access Denied"}`))
	}))
	defer server.Close()

	cfg := aws.Config{
		Region: "us-east-1",
		Credentials: aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID: "test", SecretAccessKey: "test",
			}, nil
		}),
		BaseEndpoint: aws.String(server.URL),
	}

	regions := []string{"us-east-1", "eu-west-1"}
	err := scanRegionsForClusters(context.Background(), cfg, regions)

	if err == nil {
		t.Error("expected error when all regions fail, got nil")
	}
	if !strings.Contains(err.Error(), "all regions failed") {
		t.Errorf("expected 'all regions failed' error, got: %v", err)
	}
}

func TestScanRegionsForClusters_PartialSuccess(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return empty clusters (success) for all requests using AWS API JSON format
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"clusters": []string{},
		})
	}))
	defer server.Close()

	cfg := aws.Config{
		Region: "us-east-1",
		Credentials: aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID: "test", SecretAccessKey: "test",
			}, nil
		}),
		BaseEndpoint: aws.String(server.URL),
	}

	regions := []string{"us-east-1", "eu-west-1", "ap-southeast-1"}
	err := scanRegionsForClusters(context.Background(), cfg, regions)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewAWSConfig(t *testing.T) {
	tests := []struct {
		name      string
		cred      credentials.Credential
		expectErr bool
	}{
		{
			name: "valid credentials with region",
			cred: credentials.Credential{
				Provider: "AWS",
				Details: map[string]string{
					"AccessKey":    "AKIAIOSFODNN7EXAMPLE",
					"SecretKey":    "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
					"Region":       "us-west-2",
				},
			},
			expectErr: false,
		},
		{
			name: "valid credentials without region defaults to us-east-1",
			cred: credentials.Credential{
				Provider: "AWS",
				Details: map[string]string{
					"AccessKey": "AKIAIOSFODNN7EXAMPLE",
					"SecretKey": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				},
			},
			expectErr: false,
		},
		{
			name: "valid credentials with session token",
			cred: credentials.Credential{
				Provider: "AWS",
				Details: map[string]string{
					"AccessKey":    "AKIAIOSFODNN7EXAMPLE",
					"SecretKey":    "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
					"SessionToken": "FwoGZXIvYXdzEBY",
				},
			},
			expectErr: false,
		},
		{
			name: "missing access key",
			cred: credentials.Credential{
				Provider: "AWS",
				Details:  map[string]string{"SecretKey": "secret"},
			},
			expectErr: true,
		},
		{
			name: "missing secret key",
			cred: credentials.Credential{
				Provider: "AWS",
				Details:  map[string]string{"AccessKey": "access"},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := newAWSConfig(context.Background(), tt.cred)

			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.expectErr {
				if cfg.Region == "" {
					t.Error("expected region to be set")
				}
			}
		})
	}
}
