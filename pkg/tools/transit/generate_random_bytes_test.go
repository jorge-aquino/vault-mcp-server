// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package transit

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateRandomBytesHandler_Success(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/random/16", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		jsonResponse(w, map[string]interface{}{
			"data": map[string]interface{}{
				"random_bytes": "abc==",
			},
		})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "generate_random_bytes",
			Arguments: map[string]interface{}{
				"bytes":  float64(16),
				"format": "base64",
			},
		},
	}

	result, err := generateRandomBytesHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError, "expected success, got: %s", getResultText(result))
	assert.Contains(t, getResultText(result), "abc==")
}

func TestGenerateRandomBytesHandler_Default32Bytes(t *testing.T) {
	logger := newLogger()

	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/random/32", func(w http.ResponseWriter, r *http.Request) {
		called = true
		jsonResponse(w, map[string]interface{}{
			"data": map[string]interface{}{
				"random_bytes": "defaultbytes==",
			},
		})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	// No "bytes" argument — should default to 32
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "generate_random_bytes",
			Arguments: map[string]interface{}{},
		},
	}

	result, err := generateRandomBytesHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError, "expected success, got: %s", getResultText(result))
	assert.True(t, called, "expected /v1/transit/random/32 to be called")
	assert.Contains(t, getResultText(result), "defaultbytes==")
}

func TestGenerateRandomBytesHandler_HexFormat(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/random/32", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		decodeBody(r, &body)
		assert.Equal(t, "hex", body["format"])
		jsonResponse(w, map[string]interface{}{
			"data": map[string]interface{}{
				"random_bytes": fmt.Sprintf("%x", "hexval"),
			},
		})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "generate_random_bytes",
			Arguments: map[string]interface{}{
				"format": "hex",
			},
		},
	}

	result, err := generateRandomBytesHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError, "expected success, got: %s", getResultText(result))
	assert.Contains(t, getResultText(result), "Random bytes (hex):")
}

func TestGenerateRandomBytesHandler_VaultError(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/transit/random/32", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		jsonResponse(w, map[string]interface{}{"errors": []string{"internal error"}})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "generate_random_bytes",
			Arguments: map[string]interface{}{},
		},
	}

	result, err := generateRandomBytesHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "Failed to generate random bytes")
}
