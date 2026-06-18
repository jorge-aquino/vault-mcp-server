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

func TestListTransitKeysHandler_Success(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/keys", func(w http.ResponseWriter, r *http.Request) {
		// Vault SDK sends LIST verb
		jsonResponse(w, map[string]interface{}{
			"data": map[string]interface{}{
				"keys": []interface{}{"key1", "key2"},
			},
		})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name:      "list_transit_keys",
		Arguments: map[string]interface{}{},
	}}

	result, err := listTransitKeysHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError, "expected success: %s", getResultText(result))
	text := getResultText(result)
	assert.Contains(t, text, "key1")
	assert.Contains(t, text, "key2")
}

func TestListTransitKeysHandler_Empty(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/keys", func(w http.ResponseWriter, r *http.Request) {
		// Return nil/empty — Vault returns 404 for empty list
		w.WriteHeader(http.StatusNotFound)
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name:      "list_transit_keys",
		Arguments: map[string]interface{}{},
	}}

	result, err := listTransitKeysHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	// A 404 from LIST is treated as "not found" by the Vault SDK (returns nil secret, nil err)
	assert.False(t, result.IsError, "empty list should not be an error: %s", getResultText(result))
	assert.Contains(t, getResultText(result), "No transit keys found")
}

func TestListTransitKeysHandler_VaultError(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/keys", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name:      "list_transit_keys",
		Arguments: map[string]interface{}{},
	}}

	result, err := listTransitKeysHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}
