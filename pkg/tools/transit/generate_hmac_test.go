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

func TestGenerateHMACHandler_Success(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/hmac/my-key/sha2-256", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		var body map[string]interface{}
		decodeBody(r, &body)
		// Verify input was base64-encoded
		_, err := base64.StdEncoding.DecodeString(body["input"].(string))
		require.NoError(t, err)
		jsonResponse(w, map[string]interface{}{
			"data": map[string]interface{}{
				"hmac": "vault:v1:abc123",
			},
		})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "generate_hmac",
			Arguments: map[string]interface{}{
				"name":  "my-key",
				"input": "hello world",
			},
		},
	}

	result, err := generateHMACHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError, "expected success, got: %s", getResultText(result))
	assert.Contains(t, getResultText(result), "vault:v1:abc123")
}

func TestGenerateHMACHandler_MissingName(t *testing.T) {
	logger := newLogger()

	ctx, cleanup := newTestContext(t, http.NewServeMux())
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "generate_hmac",
			Arguments: map[string]interface{}{
				"input": "hello",
			},
		},
	}

	result, err := generateHMACHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestGenerateHMACHandler_MissingInput(t *testing.T) {
	logger := newLogger()

	ctx, cleanup := newTestContext(t, http.NewServeMux())
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "generate_hmac",
			Arguments: map[string]interface{}{
				"name": "my-key",
			},
		},
	}

	result, err := generateHMACHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestGenerateHMACHandler_BadBase64(t *testing.T) {
	logger := newLogger()

	// No Vault call should be made
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/hmac/my-key/sha2-256", func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Vault should not be called for invalid base64")
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "generate_hmac",
			Arguments: map[string]interface{}{
				"name":            "my-key",
				"input":           "not valid base64!!!",
				"input_is_base64": true,
			},
		},
	}

	result, err := generateHMACHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestGenerateHMACHandler_VaultError(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/hmac/my-key/sha2-256", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		jsonResponse(w, map[string]interface{}{"errors": []string{"internal error"}})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "generate_hmac",
			Arguments: map[string]interface{}{
				"name":  "my-key",
				"input": "hello",
			},
		},
	}

	result, err := generateHMACHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}
