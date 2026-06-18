// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package pki

import (
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// createPkiIssuerHandler tests
// ---------------------------------------------------------------------------

func TestCreatePkiIssuerHandler_MissingArgs(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	// Passing nil arguments (not a map) triggers the "invalid arguments format" guard.
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name:      "create_pki_issuer",
		Arguments: nil,
	}}

	result, err := createPkiIssuerHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "invalid arguments")
}

func TestCreatePkiIssuerHandler_MissingMount(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "create_pki_issuer",
		Arguments: map[string]interface{}{
			// mount intentionally omitted
			"type":        "internal",
			"common_name": "Example CA",
			"issuer_name": "example-ca",
		},
	}}

	result, err := createPkiIssuerHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "mount")
}

func TestCreatePkiIssuerHandler_InvalidType(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "create_pki_issuer",
		Arguments: map[string]interface{}{
			"mount":       "pki",
			"type":        "external", // only "internal" is valid
			"common_name": "Example CA",
			"issuer_name": "example-ca",
		},
	}}

	result, err := createPkiIssuerHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "type")
}

func TestCreatePkiIssuerHandler_MissingCommonName(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "create_pki_issuer",
		Arguments: map[string]interface{}{
			"mount":       "pki",
			"type":        "internal",
			"issuer_name": "example-ca",
			// common_name intentionally omitted
		},
	}}

	result, err := createPkiIssuerHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "common_name")
}

func TestCreatePkiIssuerHandler_MissingIssuerName(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "create_pki_issuer",
		Arguments: map[string]interface{}{
			"mount":       "pki",
			"type":        "internal",
			"common_name": "Example CA",
			// issuer_name intentionally omitted
		},
	}}

	result, err := createPkiIssuerHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "issuer_name")
}

func TestCreatePkiIssuerHandler_MountNotFound(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	// sys/mounts returns an empty map — "pki" mount does not exist.
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, map[string]interface{}{"data": map[string]interface{}{}})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "create_pki_issuer",
		Arguments: map[string]interface{}{
			"mount":       "pki",
			"type":        "internal",
			"common_name": "Example CA",
			"issuer_name": "example-ca",
		},
	}}

	result, err := createPkiIssuerHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "does not exist")
}

func TestCreatePkiIssuerHandler_VaultWriteError(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, pkiMountsResponse("pki"))
	})
	mux.HandleFunc("/v1/pki/root/generate/internal", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "create_pki_issuer",
		Arguments: map[string]interface{}{
			"mount":       "pki",
			"type":        "internal",
			"common_name": "Example CA",
			"issuer_name": "example-ca",
		},
	}}

	result, err := createPkiIssuerHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "failed to write to path")
}

func TestCreatePkiIssuerHandler_Success(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, pkiMountsResponse("pki"))
	})
	mux.HandleFunc("/v1/pki/root/generate/internal", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		jsonResponse(w, map[string]interface{}{
			"data": map[string]interface{}{
				"certificate": "-----BEGIN CERTIFICATE-----\nMIIC...\n-----END CERTIFICATE-----",
				"issuer_id":   "abc-123",
				"issuer_name": "example-ca",
			},
		})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "create_pki_issuer",
		Arguments: map[string]interface{}{
			"mount":       "pki",
			"type":        "internal",
			"common_name": "Example CA",
			"issuer_name": "example-ca",
			"ttl":         "87600h",
		},
	}}

	result, err := createPkiIssuerHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError, "expected success: %s", getResultText(result))
	text := getResultText(result)
	assert.Contains(t, text, "example-ca")
	assert.Contains(t, text, "pki")
}

// ---------------------------------------------------------------------------
// readPkiIssuerHandler tests
// ---------------------------------------------------------------------------

func TestReadPkiIssuerHandler_MissingIssuerName(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "read_pki_issuer",
		Arguments: map[string]interface{}{
			"mount": "pki",
			// issuer_name intentionally omitted
		},
	}}

	result, err := readPkiIssuerHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "issuer_name")
}

func TestReadPkiIssuerHandler_MountNotFound(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, map[string]interface{}{"data": map[string]interface{}{}})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "read_pki_issuer",
		Arguments: map[string]interface{}{
			"mount":       "pki",
			"issuer_name": "example-ca",
		},
	}}

	result, err := readPkiIssuerHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "does not exist")
}

func TestReadPkiIssuerHandler_ListError(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, pkiMountsResponse("pki"))
	})
	// LIST returns a server error
	mux.HandleFunc("/v1/pki/issuers", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "read_pki_issuer",
		Arguments: map[string]interface{}{
			"mount":       "pki",
			"issuer_name": "example-ca",
		},
	}}

	result, err := readPkiIssuerHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "failed to read path")
}

// TestReadPkiIssuerHandler_NoIssuers exercises the "no issuers found" branch
// (LIST returns 404, which the Vault SDK treats as nil secret, nil error).
func TestReadPkiIssuerHandler_NoIssuers(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, pkiMountsResponse("pki"))
	})
	mux.HandleFunc("/v1/pki/issuers", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "read_pki_issuer",
		Arguments: map[string]interface{}{
			"mount":       "pki",
			"issuer_name": "example-ca",
		},
	}}

	result, err := readPkiIssuerHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "No issuers found")
}

// TestReadPkiIssuerHandler_MissingKeyInfo exercises the formerly-panicking
// bare type assertion path: if the LIST response lacks a "key_info" field the
// handler must return a graceful error rather than panic.
func TestReadPkiIssuerHandler_MissingKeyInfo(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, pkiMountsResponse("pki"))
	})
	// LIST succeeds but key_info is absent from the response data.
	mux.HandleFunc("/v1/pki/issuers", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, map[string]interface{}{
			"data": map[string]interface{}{
				"keys": []interface{}{"abc-123"},
				// key_info intentionally absent
			},
		})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "read_pki_issuer",
		Arguments: map[string]interface{}{
			"mount":       "pki",
			"issuer_name": "example-ca",
		},
	}}

	// Must not panic.
	result, err := readPkiIssuerHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "unexpected response format")
}

// TestReadPkiIssuerHandler_IssuerNameNotFound covers the case where key_info
// is well-formed but no entry matches the requested issuer_name.
func TestReadPkiIssuerHandler_IssuerNameNotFound(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, pkiMountsResponse("pki"))
	})
	mux.HandleFunc("/v1/pki/issuers", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, map[string]interface{}{
			"data": map[string]interface{}{
				"key_info": map[string]interface{}{
					"other-id": map[string]interface{}{
						"issuer_name": "other-ca",
					},
				},
			},
		})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "read_pki_issuer",
		Arguments: map[string]interface{}{
			"mount":       "pki",
			"issuer_name": "example-ca", // not present in key_info
		},
	}}

	result, err := readPkiIssuerHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "No issuer found with name")
}

func TestReadPkiIssuerHandler_ReadIssuerError(t *testing.T) {
	logger := newLogger()
	const issuerID = "abc-123"

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, pkiMountsResponse("pki"))
	})
	mux.HandleFunc("/v1/pki/issuers", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, map[string]interface{}{
			"data": map[string]interface{}{
				"key_info": map[string]interface{}{
					issuerID: map[string]interface{}{
						"issuer_name": "example-ca",
					},
				},
			},
		})
	})
	// The per-issuer GET returns a server error.
	mux.HandleFunc("/v1/pki/issuer/"+issuerID, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "read_pki_issuer",
		Arguments: map[string]interface{}{
			"mount":       "pki",
			"issuer_name": "example-ca",
		},
	}}

	result, err := readPkiIssuerHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "Failed to read issuer")
}

func TestReadPkiIssuerHandler_Success(t *testing.T) {
	logger := newLogger()
	const issuerID = "abc-123"

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, pkiMountsResponse("pki"))
	})
	mux.HandleFunc("/v1/pki/issuers", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, map[string]interface{}{
			"data": map[string]interface{}{
				"key_info": map[string]interface{}{
					issuerID: map[string]interface{}{
						"issuer_name": "example-ca",
					},
				},
			},
		})
	})
	mux.HandleFunc("/v1/pki/issuer/"+issuerID, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, map[string]interface{}{
			"data": map[string]interface{}{
				"issuer_id":   issuerID,
				"issuer_name": "example-ca",
				"certificate": "-----BEGIN CERTIFICATE-----\nMIIC...",
			},
		})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "read_pki_issuer",
		Arguments: map[string]interface{}{
			"mount":       "pki",
			"issuer_name": "example-ca",
		},
	}}

	result, err := readPkiIssuerHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError, "expected success: %s", getResultText(result))
	text := getResultText(result)
	assert.Contains(t, text, issuerID)
	assert.Contains(t, text, "example-ca")
}

// ---------------------------------------------------------------------------
// deletePkiIssuerHandler tests
// ---------------------------------------------------------------------------

func TestDeletePkiIssuerHandler_MissingArgs(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name:      "delete_pki_issuer",
		Arguments: nil,
	}}

	result, err := deletePkiIssuerHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "invalid arguments")
}

func TestDeletePkiIssuerHandler_MissingIssuerName(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "delete_pki_issuer",
		Arguments: map[string]interface{}{
			"mount": "pki",
			// issuer_name intentionally omitted
		},
	}}

	result, err := deletePkiIssuerHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "issuer_name")
}

// TestDeletePkiIssuerHandler_IssuerNotFound covers the case where the read
// to resolve the issuer name returns nil (issuer doesn't exist).
func TestDeletePkiIssuerHandler_IssuerNotFound(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/pki/issuer/example-ca", func(w http.ResponseWriter, r *http.Request) {
		// 404 → Vault SDK returns nil secret, nil error — triggers the err||nil guard.
		w.WriteHeader(http.StatusNotFound)
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "delete_pki_issuer",
		Arguments: map[string]interface{}{
			"mount":       "pki",
			"issuer_name": "example-ca",
		},
	}}

	result, err := deletePkiIssuerHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "not found")
}

// TestDeletePkiIssuerHandler_NoIssuerID covers the case where the read
// succeeds but issuer_id is absent from the returned data.
func TestDeletePkiIssuerHandler_NoIssuerID(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/pki/issuer/example-ca", func(w http.ResponseWriter, r *http.Request) {
		// Respond with data but no issuer_id field.
		jsonResponse(w, map[string]interface{}{
			"data": map[string]interface{}{
				"issuer_name": "example-ca",
			},
		})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "delete_pki_issuer",
		Arguments: map[string]interface{}{
			"mount":       "pki",
			"issuer_name": "example-ca",
		},
	}}

	result, err := deletePkiIssuerHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "Could not resolve issuer ID")
}

func TestDeletePkiIssuerHandler_DeleteError(t *testing.T) {
	logger := newLogger()
	const issuerID = "abc-123"

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/pki/issuer/example-ca", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, map[string]interface{}{
			"data": map[string]interface{}{
				"issuer_id":   issuerID,
				"issuer_name": "example-ca",
			},
		})
	})
	mux.HandleFunc("/v1/pki/issuer/"+issuerID, func(w http.ResponseWriter, r *http.Request) {
		// DELETE returns an error.
		w.WriteHeader(http.StatusInternalServerError)
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "delete_pki_issuer",
		Arguments: map[string]interface{}{
			"mount":       "pki",
			"issuer_name": "example-ca",
		},
	}}

	result, err := deletePkiIssuerHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "Failed to delete issuer")
}

func TestDeletePkiIssuerHandler_Success(t *testing.T) {
	logger := newLogger()
	const issuerID = "abc-123"

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/pki/issuer/example-ca", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, map[string]interface{}{
			"data": map[string]interface{}{
				"issuer_id":   issuerID,
				"issuer_name": "example-ca",
			},
		})
	})
	mux.HandleFunc("/v1/pki/issuer/"+issuerID, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusNoContent)
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "delete_pki_issuer",
		Arguments: map[string]interface{}{
			"mount":       "pki",
			"issuer_name": "example-ca",
		},
	}}

	result, err := deletePkiIssuerHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError, "expected success: %s", getResultText(result))
	text := getResultText(result)
	assert.Contains(t, text, "example-ca")
	assert.Contains(t, text, issuerID)
	assert.Contains(t, text, "pki")
}
