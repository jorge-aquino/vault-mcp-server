// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package transit

import (
	"encoding/base64"
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecryptDataHandler_Success(t *testing.T) {
	logger := newLogger()
	// base64("hello world") = "aGVsbG8gd29ybGQ="
	plainB64 := base64.StdEncoding.EncodeToString([]byte("hello world"))

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/decrypt/my-key", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		jsonResponse(w, map[string]interface{}{
			"data": map[string]interface{}{"plaintext": plainB64},
		})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "decrypt_data",
		Arguments: map[string]interface{}{
			"name":       "my-key",
			"ciphertext": "vault:v1:abc123",
		},
	}}
	result, err := decryptDataHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	text := getResultText(result)
	assert.Contains(t, text, plainB64)
	assert.Contains(t, text, "hello world")
}

func TestDecryptDataHandler_MissingName(t *testing.T) {
	logger := newLogger()
	mux := http.NewServeMux()
	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "decrypt_data",
		Arguments: map[string]interface{}{
			"ciphertext": "vault:v1:abc123",
		},
	}}
	result, err := decryptDataHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestDecryptDataHandler_MissingCiphertext(t *testing.T) {
	logger := newLogger()
	mux := http.NewServeMux()
	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "decrypt_data",
		Arguments: map[string]interface{}{
			"name": "my-key",
		},
	}}
	result, err := decryptDataHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestDecryptDataHandler_BadCiphertext(t *testing.T) {
	logger := newLogger()
	// Vault mock should NOT be called; validation rejects input before reaching Vault.
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/decrypt/my-key", func(w http.ResponseWriter, r *http.Request) {
		called = true
		jsonResponse(w, map[string]interface{}{})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "decrypt_data",
		Arguments: map[string]interface{}{
			"name":       "my-key",
			"ciphertext": "notavaultciphertext",
		},
	}}
	result, err := decryptDataHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.False(t, called, "Vault mock should not be called when validation fails")
}

func TestDecryptDataHandler_VaultError(t *testing.T) {
	logger := newLogger()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/decrypt/my-key", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		jsonResponse(w, map[string]interface{}{"errors": []string{"permission denied"}})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "decrypt_data",
		Arguments: map[string]interface{}{
			"name":       "my-key",
			"ciphertext": "vault:v1:abc123",
		},
	}}
	result, err := decryptDataHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "Failed to decrypt data")
}
