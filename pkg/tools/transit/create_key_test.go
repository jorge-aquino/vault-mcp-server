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

func TestCreateTransitKeyHandler_Success(t *testing.T) {
	logger := newLogger()
	var captured map[string]interface{}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/keys/customer-data", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		decodeBody(r, &captured)
		w.WriteHeader(http.StatusNoContent)
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name:      "create_transit_key",
		Arguments: map[string]interface{}{"name": "customer-data"},
	}}

	result, err := createTransitKeyHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError, "expected success: %s", getResultText(result))
	assert.Equal(t, "aes256-gcm96", captured["type"])
	assert.Equal(t, false, captured["exportable"])
}

func TestCreateTransitKeyHandler_CustomType(t *testing.T) {
	logger := newLogger()
	var captured map[string]interface{}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/keys/sig-key", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		decodeBody(r, &captured)
		w.WriteHeader(http.StatusNoContent)
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "create_transit_key",
		Arguments: map[string]interface{}{
			"name":       "sig-key",
			"type":       "ed25519",
			"exportable": true,
		},
	}}

	result, err := createTransitKeyHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError, "expected success: %s", getResultText(result))
	assert.Equal(t, "ed25519", captured["type"])
	assert.Equal(t, true, captured["exportable"])
}

func TestCreateTransitKeyHandler_MissingName(t *testing.T) {
	logger := newLogger()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name:      "create_transit_key",
		Arguments: map[string]interface{}{},
	}}

	// No Vault call expected, so no context needed — but we need a valid context.
	mux := http.NewServeMux()
	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	result, err := createTransitKeyHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "name")
}

func TestCreateTransitKeyHandler_InvalidKeyType(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "create_transit_key",
		Arguments: map[string]interface{}{
			"name": "bad-key",
			"type": "des3",
		},
	}}

	result, err := createTransitKeyHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "unsupported key type")
}

func TestCreateTransitKeyHandler_VaultError(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/keys/broken-key", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name:      "create_transit_key",
		Arguments: map[string]interface{}{"name": "broken-key"},
	}}

	result, err := createTransitKeyHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "Failed to create transit key")
}
