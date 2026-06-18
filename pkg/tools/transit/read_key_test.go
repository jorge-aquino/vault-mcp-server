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

func TestReadTransitKeyHandler_Success(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/keys/customer-data", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, map[string]interface{}{
			"data": map[string]interface{}{
				"name":                   "customer-data",
				"type":                   "aes256-gcm96",
				"latest_version":         float64(1),
				"min_decryption_version": float64(1),
				"exportable":             false,
				"supports_encryption":    true,
				"supports_decryption":    true,
				"supports_derivation":    true,
				"supports_signing":       false,
			},
		})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name:      "read_transit_key",
		Arguments: map[string]interface{}{"name": "customer-data"},
	}}

	result, err := readTransitKeyHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError, "expected success: %s", getResultText(result))
	text := getResultText(result)
	assert.Contains(t, text, "type")
	assert.Contains(t, text, "aes256-gcm96")
	assert.Contains(t, text, "latest_version")
}

func TestReadTransitKeyHandler_MissingName(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name:      "read_transit_key",
		Arguments: map[string]interface{}{},
	}}

	result, err := readTransitKeyHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "name")
}

func TestReadTransitKeyHandler_KeyNotFound(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/keys/missing-key", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name:      "read_transit_key",
		Arguments: map[string]interface{}{"name": "missing-key"},
	}}

	result, err := readTransitKeyHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}
