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

func TestRotateTransitKeyHandler_Success(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/keys/customer-data/rotate", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		w.WriteHeader(http.StatusNoContent)
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name:      "rotate_transit_key",
		Arguments: map[string]interface{}{"name": "customer-data"},
	}}

	result, err := rotateTransitKeyHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError, "expected success: %s", getResultText(result))
	text := getResultText(result)
	assert.Contains(t, text, "customer-data")
	assert.Contains(t, text, "rewrap_data")
}

func TestRotateTransitKeyHandler_MissingName(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name:      "rotate_transit_key",
		Arguments: map[string]interface{}{},
	}}

	result, err := rotateTransitKeyHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "name")
}

func TestRotateTransitKeyHandler_VaultError(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/keys/broken-key/rotate", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name:      "rotate_transit_key",
		Arguments: map[string]interface{}{"name": "broken-key"},
	}}

	result, err := rotateTransitKeyHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "Failed to rotate transit key")
}
