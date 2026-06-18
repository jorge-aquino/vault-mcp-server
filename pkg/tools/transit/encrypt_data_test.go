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

func TestEncryptDataHandler_Success(t *testing.T) {
	logger := newLogger()
	var capturedBody map[string]interface{}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/encrypt/customer-data", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		decodeBody(r, &capturedBody)
		jsonResponse(w, map[string]interface{}{
			"data": map[string]interface{}{
				"ciphertext": "vault:v1:abc123",
			},
		})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "encrypt_data",
		Arguments: map[string]interface{}{
			"name":      "customer-data",
			"plaintext": "hello world",
		},
	}}
	result, err := encryptDataHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError, "expected success, got error: %s", getResultText(result))

	// Verify the plaintext was auto-base64-encoded before being sent to Vault
	require.NotNil(t, capturedBody, "expected a request to Vault")
	sentPlaintext, _ := capturedBody["plaintext"].(string)
	expectedB64 := base64.StdEncoding.EncodeToString([]byte("hello world"))
	assert.Equal(t, expectedB64, sentPlaintext, "plaintext must be base64-encoded in the Vault request")

	// Verify the result contains the ciphertext
	assert.Contains(t, getResultText(result), "vault:v1:")
}

func TestEncryptDataHandler_MissingName(t *testing.T) {
	logger := newLogger()
	mux := http.NewServeMux()
	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "encrypt_data",
		Arguments: map[string]interface{}{
			"plaintext": "hello",
		},
	}}
	result, err := encryptDataHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestEncryptDataHandler_MissingPlaintext(t *testing.T) {
	logger := newLogger()
	mux := http.NewServeMux()
	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "encrypt_data",
		Arguments: map[string]interface{}{
			"name": "customer-data",
		},
	}}
	result, err := encryptDataHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestEncryptDataHandler_BadBase64WhenFlagSet(t *testing.T) {
	logger := newLogger()
	// Mock should NOT be called; validation rejects the input before reaching Vault.
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/encrypt/customer-data", func(w http.ResponseWriter, r *http.Request) {
		called = true
		jsonResponse(w, map[string]interface{}{})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "encrypt_data",
		Arguments: map[string]interface{}{
			"name":                "customer-data",
			"plaintext":           "this is not valid base64!!!",
			"plaintext_is_base64": true,
		},
	}}
	result, err := encryptDataHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.False(t, called, "Vault mock should not be called when validation fails")
}

func TestEncryptDataHandler_VaultError(t *testing.T) {
	logger := newLogger()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/encrypt/customer-data", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		jsonResponse(w, map[string]interface{}{"errors": []string{"internal server error"}})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "encrypt_data",
		Arguments: map[string]interface{}{
			"name":      "customer-data",
			"plaintext": "hello",
		},
	}}
	result, err := encryptDataHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "Failed to encrypt data")
}
