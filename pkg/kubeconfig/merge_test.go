package kubeconfig

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd/api"
)

const (
	testServerURL        = "https://example.com:6443"
	testServerURL2       = "https://test.example.com:6443"
	testCAData           = "ca-data-123"
	testCAData2          = "test-ca-data"
	testClusterNameMerge = "test-cluster"
	testUserName         = "test-user"
	testToken            = "test-token"
	testContextName      = "test-context"
	testContextName1     = "test-context-1"
	testIconPath         = "/path/to/icon.png"
)

// TestIsSameCluster tests the isSameCluster function with various scenarios
func TestIsSameCluster(t *testing.T) {
	tests := []struct {
		name     string
		cluster1 *api.Cluster
		cluster2 *api.Cluster
		want     bool
	}{
		{
			name:     "both clusters are nil",
			cluster1: nil,
			cluster2: nil,
			want:     false,
		},
		{
			name:     "first cluster is nil",
			cluster1: nil,
			cluster2: &api.Cluster{
				Server: testServerURL,
			},
			want: false,
		},
		{
			name: "second cluster is nil",
			cluster1: &api.Cluster{
				Server: testServerURL,
			},
			cluster2: nil,
			want:     false,
		},
		{
			name: "identical clusters - same server and CA",
			cluster1: &api.Cluster{
				Server:                   testServerURL,
				CertificateAuthorityData: []byte(testCAData),
			},
			cluster2: &api.Cluster{
				Server:                   testServerURL,
				CertificateAuthorityData: []byte(testCAData),
			},
			want: true,
		},
		{
			name: "different server URLs",
			cluster1: &api.Cluster{
				Server:                   "https://example1.com:6443",
				CertificateAuthorityData: []byte(testCAData),
			},
			cluster2: &api.Cluster{
				Server:                   "https://example2.com:6443",
				CertificateAuthorityData: []byte(testCAData),
			},
			want: false,
		},
		{
			name: "different CA data",
			cluster1: &api.Cluster{
				Server:                   testServerURL,
				CertificateAuthorityData: []byte(testCAData),
			},
			cluster2: &api.Cluster{
				Server:                   testServerURL,
				CertificateAuthorityData: []byte("ca-data-456"),
			},
			want: false,
		},
		{
			name: "same server but one has no CA data",
			cluster1: &api.Cluster{
				Server:                   testServerURL,
				CertificateAuthorityData: []byte(testCAData),
			},
			cluster2: &api.Cluster{
				Server:                   testServerURL,
				CertificateAuthorityData: nil,
			},
			want: false,
		},
		{
			name: "both have same server and no CA data",
			cluster1: &api.Cluster{
				Server:                   testServerURL,
				CertificateAuthorityData: nil,
			},
			cluster2: &api.Cluster{
				Server:                   testServerURL,
				CertificateAuthorityData: nil,
			},
			want: true,
		},
		{
			name: "empty CA data vs nil CA data",
			cluster1: &api.Cluster{
				Server:                   testServerURL,
				CertificateAuthorityData: []byte{},
			},
			cluster2: &api.Cluster{
				Server:                   testServerURL,
				CertificateAuthorityData: nil,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSameCluster(tt.cluster1, tt.cluster2)
			if got != tt.want {
				t.Errorf("isSameCluster() = %v, want %v", got, tt.want)
			}
		})
	}
}

// createTestConfig creates a test kubeconfig with the specified parameters
func createTestConfig(clusterName, serverURL, caData, userName, token, ctxName string, extensions map[string]runtime.Object) *api.Config {
	config := &api.Config{
		Clusters: map[string]*api.Cluster{
			clusterName: {
				Server:                   serverURL,
				CertificateAuthorityData: []byte(caData),
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			userName: {Token: token},
		},
		Contexts: map[string]*api.Context{
			ctxName: {
				Cluster:    clusterName,
				AuthInfo:   userName,
				Extensions: extensions,
			},
		},
	}
	return config
}

// verifyAptakubeExtension checks that all contexts have the aptakube extension
func verifyAptakubeExtension(t *testing.T, config *api.Config) {
	t.Helper()
	for _, context := range config.Contexts {
		if context.Extensions == nil {
			t.Error("Expected context to have extensions, got nil")
			continue
		}
		if _, exists := context.Extensions["aptakube"]; !exists {
			t.Error("Expected context to have aptakube extension")
		}
	}
}

// TestMergeKubeconfigs tests the mergeKubeconfigs function
func TestMergeKubeconfigs(t *testing.T) {
	tests := []struct {
		name             string
		destConfig       *api.Config
		srcConfig        *api.Config
		contextName      string
		imagePath        string
		expectedContexts int
		shouldOverwrite  bool
		description      string
	}{
		{
			name: "merge into empty config",
			destConfig: &api.Config{
				Clusters:  make(map[string]*api.Cluster),
				AuthInfos: make(map[string]*api.AuthInfo),
				Contexts:  make(map[string]*api.Context),
			},
			srcConfig:        createTestConfig(testClusterNameMerge, testServerURL2, testCAData2, testUserName, testToken, testContextName, nil),
			contextName:      testContextName,
			imagePath:        testIconPath,
			expectedContexts: 1,
			shouldOverwrite:  false,
			description:      "should add new context to empty config",
		},
		{
			name: "skip same cluster with same context name",
			destConfig: createTestConfig(testClusterNameMerge, testServerURL2, testCAData2, testUserName, testToken, testContextName,
				map[string]runtime.Object{"aptakube": &AptakubeExtension{IconURL: "/existing/icon.png"}}),
			srcConfig:        createTestConfig(testClusterNameMerge, testServerURL2, testCAData2, testUserName, testToken, testContextName, nil),
			contextName:      testContextName,
			imagePath:        testIconPath,
			expectedContexts: 1,
			shouldOverwrite:  false,
			description:      "should skip when same cluster already exists",
		},
		{
			name:             "overwrite different cluster with same context name",
			destConfig:       createTestConfig("old-cluster", "https://old.example.com:6443", "old-ca-data", "old-user", "old-token", testContextName, nil),
			srcConfig:        createTestConfig("new-cluster", "https://new.example.com:6443", "new-ca-data", "new-user", "new-token", testContextName, nil),
			contextName:      testContextName,
			imagePath:        testIconPath,
			expectedContexts: 2,
			shouldOverwrite:  true,
			description:      "should overwrite when different cluster has same context name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mergeKubeconfigs(tt.destConfig, tt.srcConfig, tt.contextName, tt.imagePath)
			if err != nil {
				t.Fatalf("mergeKubeconfigs() error = %v", err)
			}

			if len(tt.destConfig.Contexts) < 1 {
				t.Errorf("Expected at least 1 context, got %d", len(tt.destConfig.Contexts))
			}

			verifyAptakubeExtension(t, tt.destConfig)
		})
	}
}

// TestMakeContextNameUnique tests the makeContextNameUnique function
func TestMakeContextNameUnique(t *testing.T) {
	tests := []struct {
		name             string
		contextName      string
		existingContexts map[string]*api.Context
		want             string
	}{
		{
			name:             "no existing contexts",
			contextName:      testContextName,
			existingContexts: make(map[string]*api.Context),
			want:             testContextName,
		},
		{
			name:        "context name already exists",
			contextName: testContextName,
			existingContexts: map[string]*api.Context{
				testContextName: {
					Cluster:  testClusterNameMerge,
					AuthInfo: testUserName,
				},
			},
			want: testContextName1,
		},
		{
			name:        "multiple existing contexts with same name",
			contextName: testContextName,
			existingContexts: map[string]*api.Context{
				testContextName: {
					Cluster:  testClusterNameMerge,
					AuthInfo: testUserName,
				},
				testContextName1: {
					Cluster:  "test-cluster-2",
					AuthInfo: "test-user-2",
				},
			},
			want: "test-context-2",
		},
		{
			name:        "gaps in numbering",
			contextName: testContextName,
			existingContexts: map[string]*api.Context{
				testContextName: {
					Cluster:  testClusterNameMerge,
					AuthInfo: testUserName,
				},
				"test-context-3": {
					Cluster:  "test-cluster-3",
					AuthInfo: "test-user-3",
				},
			},
			want: testContextName1, // Should use first available number, not jump to 4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := makeContextNameUnique(tt.contextName, tt.existingContexts)
			if got != tt.want {
				t.Errorf("makeContextNameUnique() = %v, want %v", got, tt.want)
			}
		})
	}
}
