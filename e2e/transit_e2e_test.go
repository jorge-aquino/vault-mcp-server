// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

//go:build e2e

package e2e

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"testing"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTransitLifecycle exercises the full Vault Transit secrets engine lifecycle
// against a real running Vault dev server. It requires VAULT_ADDR and VAULT_TOKEN
// to be set in the environment and the transit secrets engine to be enabled at the
// default "transit/" mount path.
//
// Run with:
//
//	export VAULT_ADDR=http://127.0.0.1:8200 VAULT_TOKEN=root
//	vault secrets enable transit
//	make test-transit-e2e
func TestTransitLifecycle(t *testing.T) {
	vaultAddr := os.Getenv("VAULT_ADDR")
	vaultToken := os.Getenv("VAULT_TOKEN")
	if vaultAddr == "" || vaultToken == "" {
		t.Skip("VAULT_ADDR and VAULT_TOKEN must be set for transit e2e tests")
	}

	cfg := vaultapi.DefaultConfig()
	cfg.Address = vaultAddr
	client, err := vaultapi.NewClient(cfg)
	require.NoError(t, err)
	client.SetToken(vaultToken)

	const keyName = "e2e-test-key"

	// Best-effort cleanup: allow key deletion by updating its config first.
	t.Cleanup(func() {
		_, _ = client.Logical().Write("transit/keys/"+keyName+"/config", map[string]interface{}{
			"deletion_allowed": true,
		})
		_, _ = client.Logical().Delete("transit/keys/" + keyName)
	})

	t.Run("create_key", func(t *testing.T) {
		_, err := client.Logical().Write("transit/keys/"+keyName, map[string]interface{}{
			"type": "aes256-gcm96",
		})
		require.NoError(t, err)
	})

	t.Run("read_key", func(t *testing.T) {
		secret, err := client.Logical().Read("transit/keys/" + keyName)
		require.NoError(t, err)
		require.NotNil(t, secret)
		assert.Equal(t, "aes256-gcm96", secret.Data["type"])
		// vault/api decodes numbers as json.Number (UseNumber is set on the decoder).
		latestVersion, ok := secret.Data["latest_version"].(json.Number)
		require.True(t, ok, "expected latest_version to be json.Number, got %T", secret.Data["latest_version"])
		assert.Equal(t, "1", latestVersion.String())
	})

	const plaintext = "hello world"
	plaintextB64 := base64.StdEncoding.EncodeToString([]byte(plaintext))
	var ct1 string

	t.Run("encrypt", func(t *testing.T) {
		secret, err := client.Logical().Write("transit/encrypt/"+keyName, map[string]interface{}{
			"plaintext": plaintextB64,
		})
		require.NoError(t, err)
		require.NotNil(t, secret)
		ct1 = secret.Data["ciphertext"].(string)
		assert.True(t, strings.HasPrefix(ct1, "vault:v1:"), "expected vault:v1: prefix, got %s", ct1)
	})

	t.Run("decrypt_roundtrip", func(t *testing.T) {
		secret, err := client.Logical().Write("transit/decrypt/"+keyName, map[string]interface{}{
			"ciphertext": ct1,
		})
		require.NoError(t, err)
		require.NotNil(t, secret)
		decoded, err := base64.StdEncoding.DecodeString(secret.Data["plaintext"].(string))
		require.NoError(t, err)
		assert.Equal(t, plaintext, string(decoded))
	})

	t.Run("rotate", func(t *testing.T) {
		_, err := client.Logical().Write("transit/keys/"+keyName+"/rotate", nil)
		require.NoError(t, err)
		secret, err := client.Logical().Read("transit/keys/" + keyName)
		require.NoError(t, err)
		require.NotNil(t, secret)
		latestVersion, ok := secret.Data["latest_version"].(json.Number)
		require.True(t, ok, "expected latest_version to be json.Number, got %T", secret.Data["latest_version"])
		assert.Equal(t, "2", latestVersion.String())
	})

	var ct2 string
	t.Run("rewrap", func(t *testing.T) {
		secret, err := client.Logical().Write("transit/rewrap/"+keyName, map[string]interface{}{
			"ciphertext": ct1,
		})
		require.NoError(t, err)
		require.NotNil(t, secret)
		ct2 = secret.Data["ciphertext"].(string)
		assert.True(t, strings.HasPrefix(ct2, "vault:v2:"), "expected vault:v2: prefix, got %s", ct2)
	})

	t.Run("decrypt_rewrapped", func(t *testing.T) {
		secret, err := client.Logical().Write("transit/decrypt/"+keyName, map[string]interface{}{
			"ciphertext": ct2,
		})
		require.NoError(t, err)
		require.NotNil(t, secret)
		decoded, err := base64.StdEncoding.DecodeString(secret.Data["plaintext"].(string))
		require.NoError(t, err)
		assert.Equal(t, plaintext, string(decoded))
	})

	t.Run("hmac_and_verify", func(t *testing.T) {
		inputB64 := base64.StdEncoding.EncodeToString([]byte(plaintext))

		hmacSecret, err := client.Logical().Write("transit/hmac/"+keyName+"/sha2-256", map[string]interface{}{
			"input": inputB64,
		})
		require.NoError(t, err)
		require.NotNil(t, hmacSecret)
		hmacVal := hmacSecret.Data["hmac"].(string)
		assert.True(t, strings.HasPrefix(hmacVal, "vault:v"), "hmac should start with vault:v, got %s", hmacVal)

		// Correct input must verify as valid.
		vSecret, err := client.Logical().Write("transit/verify/"+keyName+"/sha2-256", map[string]interface{}{
			"input": inputB64,
			"hmac":  hmacVal,
		})
		require.NoError(t, err)
		require.NotNil(t, vSecret)
		assert.Equal(t, true, vSecret.Data["valid"])

		// Tampered input must not verify.
		tamperedB64 := base64.StdEncoding.EncodeToString([]byte("tampered"))
		vSecret2, err := client.Logical().Write("transit/verify/"+keyName+"/sha2-256", map[string]interface{}{
			"input": tamperedB64,
			"hmac":  hmacVal,
		})
		require.NoError(t, err)
		require.NotNil(t, vSecret2)
		assert.Equal(t, false, vSecret2.Data["valid"])
	})
}
