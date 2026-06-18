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

func TestVerifyHMACHandler_ValidTrue(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/verify/my-key/sha2-256", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		jsonResponse(w, map[string]interface{}{
			"data": map[string]interface{}{
				"valid": true,
			},
		})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "verify_hmac",
			Arguments: map[string]interface{}{
				"name":  "my-key",
				"input": "hello world",
				"hmac":  "vault:v1:abc123",
			},
		},
	}

	result, err := verifyHMACHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError, "expected success, got: %s", getResultText(result))
	assert.Contains(t, getResultText(result), "valid: true")
}

func TestVerifyHMACHandler_ValidFalse(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/verify/my-key/sha2-256", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, map[string]interface{}{
			"data": map[string]interface{}{
				"valid": false,
			},
		})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "verify_hmac",
			Arguments: map[string]interface{}{
				"name":  "my-key",
				"input": "hello world",
				"hmac":  "vault:v1:wronghmac",
			},
		},
	}

	result, err := verifyHMACHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError, "expected success, got: %s", getResultText(result))
	assert.Contains(t, getResultText(result), "valid: false")
}

func TestVerifyHMACHandler_MissingName(t *testing.T) {
	logger := newLogger()

	ctx, cleanup := newTestContext(t, http.NewServeMux())
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "verify_hmac",
			Arguments: map[string]interface{}{
				"input": "hello",
				"hmac":  "vault:v1:abc",
			},
		},
	}

	result, err := verifyHMACHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestVerifyHMACHandler_MissingInput(t *testing.T) {
	logger := newLogger()

	ctx, cleanup := newTestContext(t, http.NewServeMux())
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "verify_hmac",
			Arguments: map[string]interface{}{
				"name": "my-key",
				"hmac": "vault:v1:abc",
			},
		},
	}

	result, err := verifyHMACHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestVerifyHMACHandler_MissingHMAC(t *testing.T) {
	logger := newLogger()

	ctx, cleanup := newTestContext(t, http.NewServeMux())
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "verify_hmac",
			Arguments: map[string]interface{}{
				"name":  "my-key",
				"input": "hello",
			},
		},
	}

	result, err := verifyHMACHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestVerifyHMACHandler_VaultError(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/verify/my-key/sha2-256", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		jsonResponse(w, map[string]interface{}{"errors": []string{"internal error"}})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "verify_hmac",
			Arguments: map[string]interface{}{
				"name":  "my-key",
				"input": "hello",
				"hmac":  "vault:v1:abc",
			},
		},
	}

	result, err := verifyHMACHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}
