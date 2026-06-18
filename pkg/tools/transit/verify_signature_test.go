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

func TestVerifySignatureHandler_ValidTrue(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/verify/my-key/sha2-256", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		var body map[string]interface{}
		decodeBody(r, &body)
		assert.Equal(t, "vault:v1:sig123", body["signature"])
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
			Name: "verify_signature",
			Arguments: map[string]interface{}{
				"name":      "my-key",
				"input":     "hello world",
				"signature": "vault:v1:sig123",
			},
		},
	}

	result, err := verifySignatureHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError, "expected success, got: %s", getResultText(result))
	assert.Contains(t, getResultText(result), "valid: true")
}

func TestVerifySignatureHandler_ValidFalse(t *testing.T) {
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
			Name: "verify_signature",
			Arguments: map[string]interface{}{
				"name":      "my-key",
				"input":     "hello world",
				"signature": "vault:v1:badsig",
			},
		},
	}

	result, err := verifySignatureHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError, "expected success, got: %s", getResultText(result))
	assert.Contains(t, getResultText(result), "valid: false")
}

func TestVerifySignatureHandler_MissingName(t *testing.T) {
	logger := newLogger()

	ctx, cleanup := newTestContext(t, http.NewServeMux())
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "verify_signature",
			Arguments: map[string]interface{}{
				"input":     "hello",
				"signature": "vault:v1:sig",
			},
		},
	}

	result, err := verifySignatureHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestVerifySignatureHandler_MissingInput(t *testing.T) {
	logger := newLogger()

	ctx, cleanup := newTestContext(t, http.NewServeMux())
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "verify_signature",
			Arguments: map[string]interface{}{
				"name":      "my-key",
				"signature": "vault:v1:sig",
			},
		},
	}

	result, err := verifySignatureHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestVerifySignatureHandler_MissingSignature(t *testing.T) {
	logger := newLogger()

	ctx, cleanup := newTestContext(t, http.NewServeMux())
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "verify_signature",
			Arguments: map[string]interface{}{
				"name":  "my-key",
				"input": "hello",
			},
		},
	}

	result, err := verifySignatureHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}
