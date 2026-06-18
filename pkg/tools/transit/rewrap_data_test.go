// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package transit

import (
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRewrapDataHandler_Success(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/rewrap/customer-data", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		jsonResponse(w, map[string]interface{}{
			"data": map[string]interface{}{
				"ciphertext": "vault:v2:newciphertext",
			},
		})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "rewrap_data",
		Arguments: map[string]interface{}{
			"name":       "customer-data",
			"ciphertext": "vault:v1:oldciphertext",
		},
	}}
	result, err := rewrapDataHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.False(t, result.IsError, "expected success, got error: %s", getResultText(result))
	assert.Contains(t, getResultText(result), "vault:v2:")
}

func TestRewrapDataHandler_BadCiphertext(t *testing.T) {
	logger := newLogger()
	// Vault mock should NOT be called; validation rejects input before reaching Vault.
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/rewrap/customer-data", func(w http.ResponseWriter, r *http.Request) {
		called = true
		jsonResponse(w, map[string]interface{}{})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "rewrap_data",
		Arguments: map[string]interface{}{
			"name":       "customer-data",
			"ciphertext": "notavaultciphertext",
		},
	}}
	result, err := rewrapDataHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.False(t, called, "Vault mock should not be called when validation fails")
}

func TestRewrapDataHandler_MissingName(t *testing.T) {
	logger := newLogger()
	mux := http.NewServeMux()
	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "rewrap_data",
		Arguments: map[string]interface{}{
			"ciphertext": "vault:v1:abc123",
		},
	}}
	result, err := rewrapDataHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestRewrapDataHandler_VaultError(t *testing.T) {
	logger := newLogger()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/rewrap/customer-data", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		jsonResponse(w, map[string]interface{}{"errors": []string{"internal server error"}})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "rewrap_data",
		Arguments: map[string]interface{}{
			"name":       "customer-data",
			"ciphertext": "vault:v1:oldciphertext",
		},
	}}
	result, err := rewrapDataHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "Failed to rewrap data")
}
