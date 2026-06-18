// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package transit

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/hashicorp/vault/api"
)

// DefaultMount is the conventional Transit mount path.
const DefaultMount = "transit"

// transitPath builds a Vault Transit API path, e.g. transitPath("transit","keys","demo")
// -> "transit/keys/demo". segment is one of: keys, encrypt, decrypt, rewrap, hmac, verify,
// sign, hash, random. When name is empty, the trailing slash/name is omitted.
func transitPath(mount, segment, name string) string {
	mount = strings.Trim(mount, "/")
	if name == "" {
		return fmt.Sprintf("%s/%s", mount, segment)
	}
	return fmt.Sprintf("%s/%s/%s", mount, segment, name)
}

// resolveMount returns the provided mount or DefaultMount when empty.
func resolveMount(args map[string]interface{}) string {
	if m, ok := args["mount"].(string); ok && strings.TrimSpace(m) != "" {
		return strings.Trim(m, "/")
	}
	return DefaultMount
}

// extractString returns a string arg; errors if required and missing/empty.
func extractString(args map[string]interface{}, key string, required bool) (string, error) {
	v, ok := args[key]
	if !ok || v == nil {
		if required {
			return "", fmt.Errorf("missing required parameter '%s'", key)
		}
		return "", nil
	}
	s, ok := v.(string)
	if !ok {
		if required {
			return "", fmt.Errorf("parameter '%s' must be a string", key)
		}
		return "", nil
	}
	if required && strings.TrimSpace(s) == "" {
		return "", fmt.Errorf("parameter '%s' must not be empty", key)
	}
	return s, nil
}

// extractBool performs optional typed extraction with a default value.
func extractBool(args map[string]interface{}, key string, def bool) bool {
	v, ok := args[key]
	if !ok || v == nil {
		return def
	}
	b, ok := v.(bool)
	if !ok {
		return def
	}
	return b
}

// extractInt performs optional typed extraction with a default value.
func extractInt(args map[string]interface{}, key string, def int) int {
	v, ok := args[key]
	if !ok || v == nil {
		return def
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return def
}

// validateKeyName guards against empty/whitespace/path-traversal in key names.
func validateKeyName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("key name must not be empty")
	}
	if strings.Contains(name, "/") || strings.Contains(name, "..") {
		return fmt.Errorf("key name must not contain '/' or '..'")
	}
	return nil
}

// validateBase64 ensures s is valid standard base64.
func validateBase64(s string) error {
	if _, err := base64.StdEncoding.DecodeString(s); err != nil {
		return fmt.Errorf("value is not valid base64: %w", err)
	}
	return nil
}

// validateCiphertext ensures Vault ciphertext format: must start with "vault:v".
func validateCiphertext(ct string) error {
	if !strings.HasPrefix(ct, "vault:v") {
		return fmt.Errorf("invalid ciphertext: expected a 'vault:v<version>:...' value")
	}
	return nil
}

// dataString safely pulls a string field out of a *api.Secret response.
func dataString(secret *api.Secret, key string) (string, error) {
	if secret == nil || secret.Data == nil {
		return "", fmt.Errorf("empty response from Vault")
	}
	v, ok := secret.Data[key]
	if !ok || v == nil {
		return "", fmt.Errorf("key '%s' not found in Vault response", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("key '%s' in Vault response is not a string", key)
	}
	return s, nil
}
