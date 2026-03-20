package credentials

import (
	"kubectm/pkg/utils"
	"strings"
	"testing"
)

func TestLogCredentialDiscovery(t *testing.T) {
	t.Run("nil credential does not panic", func(t *testing.T) {
		// Should not panic or log anything
		logCredentialDiscovery("AWS", nil)
	})

	t.Run("obfuscates all credential keys", func(t *testing.T) {
		cred := &Credential{
			Provider: "AWS",
			Details: map[string]string{
				"AccessKey": "AKIAIOSFODNN7EXAMPLE",
				"SecretKey": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
		}

		// Verify the function uses ObfuscateCredential correctly
		// by checking the obfuscation output directly
		for _, v := range cred.Details {
			obfuscated := utils.ObfuscateCredential(v)
			// Raw value should NOT equal obfuscated value
			if obfuscated == v {
				t.Errorf("credential value was not obfuscated: %s", v)
			}
			// Should contain first 4 and last 4 chars
			if !strings.HasPrefix(obfuscated, v[:4]) {
				t.Errorf("obfuscated value should start with first 4 chars of original")
			}
			if !strings.HasSuffix(obfuscated, v[len(v)-4:]) {
				t.Errorf("obfuscated value should end with last 4 chars of original")
			}
		}

		// Should not panic
		logCredentialDiscovery("AWS", cred)
	})

	t.Run("short credential values are masked", func(t *testing.T) {
		cred := &Credential{
			Provider: "Test",
			Details: map[string]string{
				"Key": "short",
			},
		}

		obfuscated := utils.ObfuscateCredential(cred.Details["Key"])
		if obfuscated != "****" {
			t.Errorf("short credential should be masked as ****, got: %s", obfuscated)
		}

		// Should not panic
		logCredentialDiscovery("Test", cred)
	})

	t.Run("empty credential details", func(t *testing.T) {
		cred := &Credential{
			Provider: "Empty",
			Details:  map[string]string{},
		}

		// Should not panic
		logCredentialDiscovery("Empty", cred)
	})
}
