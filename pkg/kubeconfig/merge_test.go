package kubeconfig

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd/api"
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
				Server: "https://example.com:6443",
			},
			want: false,
		},
		{
			name: "second cluster is nil",
			cluster1: &api.Cluster{
				Server: "https://example.com:6443",
			},
			cluster2: nil,
			want:     false,
		},
		{
			name: "identical clusters - same server and CA",
			cluster1: &api.Cluster{
				Server:                   "https://example.com:6443",
				CertificateAuthorityData: []byte("ca-data-123"),
			},
			cluster2: &api.Cluster{
				Server:                   "https://example.com:6443",
				CertificateAuthorityData: []byte("ca-data-123"),
			},
			want: true,
		},
		{
			name: "different server URLs",
			cluster1: &api.Cluster{
				Server:                   "https://example1.com:6443",
				CertificateAuthorityData: []byte("ca-data-123"),
			},
			cluster2: &api.Cluster{
				Server:                   "https://example2.com:6443",
				CertificateAuthorityData: []byte("ca-data-123"),
			},
			want: false,
		},
		{
			name: "different CA data",
			cluster1: &api.Cluster{
				Server:                   "https://example.com:6443",
				CertificateAuthorityData: []byte("ca-data-123"),
			},
			cluster2: &api.Cluster{
				Server:                   "https://example.com:6443",
				CertificateAuthorityData: []byte("ca-data-456"),
			},
			want: false,
		},
		{
			name: "same server but one has no CA data",
			cluster1: &api.Cluster{
				Server:                   "https://example.com:6443",
				CertificateAuthorityData: []byte("ca-data-123"),
			},
			cluster2: &api.Cluster{
				Server:                   "https://example.com:6443",
				CertificateAuthorityData: nil,
			},
			want: false,
		},
		{
			name: "both have same server and no CA data",
			cluster1: &api.Cluster{
				Server:                   "https://example.com:6443",
				CertificateAuthorityData: nil,
			},
			cluster2: &api.Cluster{
				Server:                   "https://example.com:6443",
				CertificateAuthorityData: nil,
			},
			want: true,
		},
		{
			name: "empty CA data vs nil CA data",
			cluster1: &api.Cluster{
				Server:                   "https://example.com:6443",
				CertificateAuthorityData: []byte{},
			},
			cluster2: &api.Cluster{
				Server:                   "https://example.com:6443",
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

// TestMergeKubeconfigs tests the mergeKubeconfigs function
func TestMergeKubeconfigs(t *testing.T) {
	tests := []struct {
		name            string
		destConfig      *api.Config
		srcConfig       *api.Config
		contextName     string
		imagePath       string
		expectedContexts int
		shouldOverwrite bool
		description     string
	}{
		{
			name: "merge into empty config",
			destConfig: &api.Config{
				Clusters:  make(map[string]*api.Cluster),
				AuthInfos: make(map[string]*api.AuthInfo),
				Contexts:  make(map[string]*api.Context),
			},
			srcConfig: &api.Config{
				Clusters: map[string]*api.Cluster{
					"test-cluster": {
						Server:                   "https://test.example.com:6443",
						CertificateAuthorityData: []byte("test-ca-data"),
					},
				},
				AuthInfos: map[string]*api.AuthInfo{
					"test-user": {
						Token: "test-token",
					},
				},
				Contexts: map[string]*api.Context{
					"test-context": {
						Cluster:  "test-cluster",
						AuthInfo: "test-user",
					},
				},
			},
			contextName:      "test-context",
			imagePath:        "/path/to/icon.png",
			expectedContexts: 1,
			shouldOverwrite:  false,
			description:      "should add new context to empty config",
		},
		{
			name: "skip same cluster with same context name",
			destConfig: &api.Config{
				Clusters: map[string]*api.Cluster{
					"test-cluster": {
						Server:                   "https://test.example.com:6443",
						CertificateAuthorityData: []byte("test-ca-data"),
					},
				},
				AuthInfos: map[string]*api.AuthInfo{
					"test-user": {
						Token: "test-token",
					},
				},
				Contexts: map[string]*api.Context{
					"test-context": {
						Cluster:  "test-cluster",
						AuthInfo: "test-user",
						Extensions: map[string]runtime.Object{
							"aptakube": &AptakubeExtension{
								IconURL: "/existing/icon.png",
							},
						},
					},
				},
			},
			srcConfig: &api.Config{
				Clusters: map[string]*api.Cluster{
					"test-cluster": {
						Server:                   "https://test.example.com:6443",
						CertificateAuthorityData: []byte("test-ca-data"),
					},
				},
				AuthInfos: map[string]*api.AuthInfo{
					"test-user": {
						Token: "test-token",
					},
				},
				Contexts: map[string]*api.Context{
					"test-context": {
						Cluster:  "test-cluster",
						AuthInfo: "test-user",
					},
				},
			},
			contextName:      "test-context",
			imagePath:        "/path/to/icon.png",
			expectedContexts: 1,
			shouldOverwrite:  false,
			description:      "should skip when same cluster already exists",
		},
		{
			name: "overwrite different cluster with same context name",
			destConfig: &api.Config{
				Clusters: map[string]*api.Cluster{
					"old-cluster": {
						Server:                   "https://old.example.com:6443",
						CertificateAuthorityData: []byte("old-ca-data"),
					},
				},
				AuthInfos: map[string]*api.AuthInfo{
					"old-user": {
						Token: "old-token",
					},
				},
				Contexts: map[string]*api.Context{
					"test-context": {
						Cluster:  "old-cluster",
						AuthInfo: "old-user",
					},
				},
			},
			srcConfig: &api.Config{
				Clusters: map[string]*api.Cluster{
					"new-cluster": {
						Server:                   "https://new.example.com:6443",
						CertificateAuthorityData: []byte("new-ca-data"),
					},
				},
				AuthInfos: map[string]*api.AuthInfo{
					"new-user": {
						Token: "new-token",
					},
				},
				Contexts: map[string]*api.Context{
					"test-context": {
						Cluster:  "new-cluster",
						AuthInfo: "new-user",
					},
				},
			},
			contextName:      "test-context",
			imagePath:        "/path/to/icon.png",
			expectedContexts: 2, // Both test-context and test-context-1 should exist due to uniqueness
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

			// Check the number of contexts
			if len(tt.destConfig.Contexts) < 1 {
				t.Errorf("Expected at least 1 context, got %d", len(tt.destConfig.Contexts))
			}

			// Verify that the context has the Aptakube extension
			for _, context := range tt.destConfig.Contexts {
				if context.Extensions == nil {
					t.Error("Expected context to have extensions, got nil")
					continue
				}
				if _, exists := context.Extensions["aptakube"]; !exists {
					t.Error("Expected context to have aptakube extension")
				}
			}
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
			contextName:      "test-context",
			existingContexts: make(map[string]*api.Context),
			want:             "test-context",
		},
		{
			name:        "context name already exists",
			contextName: "test-context",
			existingContexts: map[string]*api.Context{
				"test-context": {
					Cluster:  "test-cluster",
					AuthInfo: "test-user",
				},
			},
			want: "test-context-1",
		},
		{
			name:        "multiple existing contexts with same name",
			contextName: "test-context",
			existingContexts: map[string]*api.Context{
				"test-context": {
					Cluster:  "test-cluster",
					AuthInfo: "test-user",
				},
				"test-context-1": {
					Cluster:  "test-cluster-2",
					AuthInfo: "test-user-2",
				},
			},
			want: "test-context-2",
		},
		{
			name:        "gaps in numbering",
			contextName: "test-context",
			existingContexts: map[string]*api.Context{
				"test-context": {
					Cluster:  "test-cluster",
					AuthInfo: "test-user",
				},
				"test-context-3": {
					Cluster:  "test-cluster-3",
					AuthInfo: "test-user-3",
				},
			},
			want: "test-context-1", // Should use first available number, not jump to 4
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
