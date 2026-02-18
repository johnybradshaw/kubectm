package credentials

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRetrieveAWSCredentialsFromEnv(t *testing.T) {
	// Save and clear all AWS env vars
	origAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	origSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	origSessionToken := os.Getenv("AWS_SESSION_TOKEN")
	origRegion := os.Getenv("AWS_DEFAULT_REGION")
	origProfile := os.Getenv("AWS_PROFILE")
	defer func() {
		os.Setenv("AWS_ACCESS_KEY_ID", origAccessKey)
		os.Setenv("AWS_SECRET_ACCESS_KEY", origSecretKey)
		os.Setenv("AWS_SESSION_TOKEN", origSessionToken)
		os.Setenv("AWS_DEFAULT_REGION", origRegion)
		os.Setenv("AWS_PROFILE", origProfile)
	}()

	tests := []struct {
		name           string
		accessKey      string
		secretKey      string
		sessionToken   string
		region         string
		expectNil      bool
		expectError    bool
		checkToken     bool
		checkRegion    bool
	}{
		{
			name:        "both env vars set",
			accessKey:   "AKIAIOSFODNN7EXAMPLE",
			secretKey:   "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			expectNil:   false,
			expectError: false,
		},
		{
			name:        "with session token and region",
			accessKey:   "AKIAIOSFODNN7EXAMPLE",
			secretKey:   "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			sessionToken: "FwoGZXIvYXdzEBY",
			region:      "us-east-1",
			expectNil:   false,
			expectError: false,
			checkToken:  true,
			checkRegion: true,
		},
		{
			name:        "only access key set",
			accessKey:   "AKIAIOSFODNN7EXAMPLE",
			secretKey:   "",
			expectNil:   true,
			expectError: false,
		},
		{
			name:        "only secret key set",
			accessKey:   "",
			secretKey:   "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			expectNil:   true,
			expectError: false,
		},
		{
			name:        "neither set",
			accessKey:   "",
			secretKey:   "",
			expectNil:   true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set env vars for this test case
			os.Setenv("AWS_ACCESS_KEY_ID", tt.accessKey)
			os.Setenv("AWS_SECRET_ACCESS_KEY", tt.secretKey)
			os.Setenv("AWS_SESSION_TOKEN", tt.sessionToken)
			os.Setenv("AWS_DEFAULT_REGION", tt.region)

			// Point HOME to a non-existent dir so file fallback returns nil
			origHome := os.Getenv("HOME")
			os.Setenv("HOME", t.TempDir())
			defer os.Setenv("HOME", origHome)

			cred, err := retrieveAWSCredentials()

			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.expectNil {
				if cred != nil {
					t.Errorf("expected nil credential, got %+v", cred)
				}
				return
			}

			if cred == nil {
				t.Fatal("expected credential, got nil")
			}
			if cred.Provider != "AWS" {
				t.Errorf("expected provider AWS, got %s", cred.Provider)
			}
			if cred.Details["AccessKey"] != tt.accessKey {
				t.Errorf("expected access key %s, got %s", tt.accessKey, cred.Details["AccessKey"])
			}
			if cred.Details["SecretKey"] != tt.secretKey {
				t.Errorf("expected secret key %s, got %s", tt.secretKey, cred.Details["SecretKey"])
			}
			if tt.checkToken {
				if cred.Details["SessionToken"] != tt.sessionToken {
					t.Errorf("expected session token %s, got %s", tt.sessionToken, cred.Details["SessionToken"])
				}
			}
			if tt.checkRegion {
				if cred.Details["Region"] != tt.region {
					t.Errorf("expected region %s, got %s", tt.region, cred.Details["Region"])
				}
			}
		})
	}
}

func TestParseAWSCredentialsFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		profile     string
		expectNil   bool
		accessKey   string
		secretKey   string
		sessionToken string
	}{
		{
			name: "default profile",
			content: `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
`,
			profile:   "default",
			expectNil: false,
			accessKey: "AKIAIOSFODNN7EXAMPLE",
			secretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		},
		{
			name: "named profile",
			content: `[default]
aws_access_key_id = DEFAULT_KEY
aws_secret_access_key = DEFAULT_SECRET

[production]
aws_access_key_id = PROD_KEY_EXAMPLE1234
aws_secret_access_key = PROD_SECRET_EXAMPLE1234567890
`,
			profile:   "production",
			expectNil: false,
			accessKey: "PROD_KEY_EXAMPLE1234",
			secretKey: "PROD_SECRET_EXAMPLE1234567890",
		},
		{
			name: "profile with session token",
			content: `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
aws_session_token = FwoGZXIvYXdzEBYaDH
`,
			profile:      "default",
			expectNil:    false,
			accessKey:    "AKIAIOSFODNN7EXAMPLE",
			secretKey:    "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			sessionToken: "FwoGZXIvYXdzEBYaDH",
		},
		{
			name: "profile not found",
			content: `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
`,
			profile:   "nonexistent",
			expectNil: true,
		},
		{
			name:      "empty file",
			content:   "",
			profile:   "default",
			expectNil: true,
		},
		{
			name: "missing secret key",
			content: `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
`,
			profile:   "default",
			expectNil: true,
		},
		{
			name: "comments and blank lines",
			content: `# This is a comment
; This is also a comment

[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
# inline comment line
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
`,
			profile:   "default",
			expectNil: false,
			accessKey: "AKIAIOSFODNN7EXAMPLE",
			secretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		},
		{
			name: "values with spaces around equals",
			content: `[default]
aws_access_key_id  =  AKIAIOSFODNN7EXAMPLE
aws_secret_access_key  =  wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
`,
			profile:   "default",
			expectNil: false,
			accessKey: "AKIAIOSFODNN7EXAMPLE",
			secretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cred := parseAWSCredentialsFile([]byte(tt.content), tt.profile)

			if tt.expectNil {
				if cred != nil {
					t.Errorf("expected nil, got %+v", cred)
				}
				return
			}

			if cred == nil {
				t.Fatal("expected credential, got nil")
			}
			if cred.Provider != "AWS" {
				t.Errorf("expected provider AWS, got %s", cred.Provider)
			}
			if cred.Details["AccessKey"] != tt.accessKey {
				t.Errorf("expected access key %s, got %s", tt.accessKey, cred.Details["AccessKey"])
			}
			if cred.Details["SecretKey"] != tt.secretKey {
				t.Errorf("expected secret key %s, got %s", tt.secretKey, cred.Details["SecretKey"])
			}
			if tt.sessionToken != "" {
				if cred.Details["SessionToken"] != tt.sessionToken {
					t.Errorf("expected session token %s, got %s", tt.sessionToken, cred.Details["SessionToken"])
				}
			}
		})
	}
}

func TestRetrieveAWSCredentialsFromFile(t *testing.T) {
	// Clear env vars so file fallback is used
	origAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	origSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	origProfile := os.Getenv("AWS_PROFILE")
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("AWS_ACCESS_KEY_ID", origAccessKey)
		os.Setenv("AWS_SECRET_ACCESS_KEY", origSecretKey)
		os.Setenv("AWS_PROFILE", origProfile)
		os.Setenv("HOME", origHome)
	}()

	os.Setenv("AWS_ACCESS_KEY_ID", "")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "")
	os.Setenv("AWS_PROFILE", "")

	// Create a temp home directory with an AWS credentials file
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)

	awsDir := filepath.Join(tempDir, ".aws")
	if err := os.MkdirAll(awsDir, 0700); err != nil {
		t.Fatalf("failed to create .aws directory: %v", err)
	}

	credContent := `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
region = us-west-2
`
	credFile := filepath.Join(awsDir, "credentials")
	if err := os.WriteFile(credFile, []byte(credContent), 0600); err != nil {
		t.Fatalf("failed to write credentials file: %v", err)
	}

	cred, err := retrieveAWSCredentials()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cred == nil {
		t.Fatal("expected credential, got nil")
	}
	if cred.Provider != "AWS" {
		t.Errorf("expected provider AWS, got %s", cred.Provider)
	}
	if cred.Details["AccessKey"] != "AKIAIOSFODNN7EXAMPLE" {
		t.Errorf("expected access key AKIAIOSFODNN7EXAMPLE, got %s", cred.Details["AccessKey"])
	}
	if cred.Details["Region"] != "us-west-2" {
		t.Errorf("expected region us-west-2, got %s", cred.Details["Region"])
	}
}

func TestRetrieveAWSCredentialsWithProfile(t *testing.T) {
	origAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	origSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	origProfile := os.Getenv("AWS_PROFILE")
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("AWS_ACCESS_KEY_ID", origAccessKey)
		os.Setenv("AWS_SECRET_ACCESS_KEY", origSecretKey)
		os.Setenv("AWS_PROFILE", origProfile)
		os.Setenv("HOME", origHome)
	}()

	os.Setenv("AWS_ACCESS_KEY_ID", "")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "")
	os.Setenv("AWS_PROFILE", "staging")

	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)

	awsDir := filepath.Join(tempDir, ".aws")
	if err := os.MkdirAll(awsDir, 0700); err != nil {
		t.Fatalf("failed to create .aws directory: %v", err)
	}

	credContent := `[default]
aws_access_key_id = DEFAULT_KEY_1234567890
aws_secret_access_key = DEFAULT_SECRET_1234567890

[staging]
aws_access_key_id = STAGING_KEY_1234567890
aws_secret_access_key = STAGING_SECRET_1234567890
`
	credFile := filepath.Join(awsDir, "credentials")
	if err := os.WriteFile(credFile, []byte(credContent), 0600); err != nil {
		t.Fatalf("failed to write credentials file: %v", err)
	}

	cred, err := retrieveAWSCredentials()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cred == nil {
		t.Fatal("expected credential, got nil")
	}
	if cred.Details["AccessKey"] != "STAGING_KEY_1234567890" {
		t.Errorf("expected staging access key, got %s", cred.Details["AccessKey"])
	}
}
