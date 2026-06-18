// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package pki

// Tests for issuePkiCertificateHandler and revokePkiCertificateHandler.
// Shared test helpers (fakeSession, newLogger, newTestContext, jsonResponse,
// getResultText) live in pki_test.go; pkiMountsResponse lives in pki_role_test.go.

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// issuePkiCertificateHandler tests
// ---------------------------------------------------------------------------

func TestIssuePkiCertificateHandler_Success(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, pkiMountsResponse("pki"))
	})
	mux.HandleFunc("/v1/pki/issue/web-server", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		jsonResponse(w, map[string]interface{}{
			"data": map[string]interface{}{
				"certificate":  "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----",
				"serial_number": "40:1d:4a:bb:18:03:64:a9:00:c3:64:43:12:1f:9a:41",
				"expiration":   1893456000,
			},
		})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "issue_pki_certificate",
			Arguments: map[string]interface{}{
				"mount":       "pki",
				"role_name":   "web-server",
				"common_name": "example.com",
			},
		},
	}

	result, err := issuePkiCertificateHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError, "expected success, got: %s", getResultText(result))
	assert.Contains(t, getResultText(result), "serial_number")
}

func TestIssuePkiCertificateHandler_MissingRoleName(t *testing.T) {
	logger := newLogger()

	ctx, cleanup := newTestContext(t, http.NewServeMux())
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "issue_pki_certificate",
			Arguments: map[string]interface{}{
				"mount":       "pki",
				"common_name": "example.com",
			},
		},
	}

	result, err := issuePkiCertificateHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "role_name")
}

func TestIssuePkiCertificateHandler_MissingCommonName(t *testing.T) {
	logger := newLogger()

	ctx, cleanup := newTestContext(t, http.NewServeMux())
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "issue_pki_certificate",
			Arguments: map[string]interface{}{
				"mount":     "pki",
				"role_name": "web-server",
			},
		},
	}

	result, err := issuePkiCertificateHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "common_name")
}

func TestIssuePkiCertificateHandler_MountNotFound(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	// The mounts response does not include the requested mount.
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, pkiMountsResponse("other-pki"))
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "issue_pki_certificate",
			Arguments: map[string]interface{}{
				"mount":       "pki",
				"role_name":   "web-server",
				"common_name": "example.com",
			},
		},
	}

	result, err := issuePkiCertificateHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "pki")
}

func TestIssuePkiCertificateHandler_VaultAPIError(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, pkiMountsResponse("pki"))
	})
	mux.HandleFunc("/v1/pki/issue/web-server", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		jsonResponse(w, map[string]interface{}{"errors": []string{"internal server error"}})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "issue_pki_certificate",
			Arguments: map[string]interface{}{
				"mount":       "pki",
				"role_name":   "web-server",
				"common_name": "example.com",
			},
		},
	}

	result, err := issuePkiCertificateHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "failed to write to path")
}

// ---------------------------------------------------------------------------
// revokePkiCertificateHandler tests
// ---------------------------------------------------------------------------

func TestRevokePkiCertificateHandler_Success(t *testing.T) {
	logger := newLogger()
	const serial = "40:1d:4a:bb:18:03:64:a9:00:c3:64:43:12:1f:9a:41"

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/pki/revoke", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
		assert.Equal(t, serial, body["serial_number"])
		jsonResponse(w, map[string]interface{}{
			"data": map[string]interface{}{
				"revocation_time_rfc3339": "2025-01-18T12:00:00Z",
			},
		})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "revoke_pki_certificate",
			Arguments: map[string]interface{}{
				"mount":         "pki",
				"serial_number": serial,
			},
		},
	}

	result, err := revokePkiCertificateHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError, "expected success, got: %s", getResultText(result))
	assert.Contains(t, getResultText(result), "Successfully revoked certificate")
	assert.Contains(t, getResultText(result), serial)
	assert.Contains(t, getResultText(result), "2025-01-18T12:00:00Z")
}

func TestRevokePkiCertificateHandler_MissingSerialNumber(t *testing.T) {
	logger := newLogger()

	ctx, cleanup := newTestContext(t, http.NewServeMux())
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "revoke_pki_certificate",
			Arguments: map[string]interface{}{
				"mount": "pki",
			},
		},
	}

	result, err := revokePkiCertificateHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "serial_number")
}

func TestRevokePkiCertificateHandler_VaultAPIError(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/pki/revoke", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		jsonResponse(w, map[string]interface{}{"errors": []string{"permission denied"}})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "revoke_pki_certificate",
			Arguments: map[string]interface{}{
				"mount":         "pki",
				"serial_number": "40:1d:4a:bb",
			},
		},
	}

	result, err := revokePkiCertificateHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "Failed to revoke certificate")
}

func TestRevokePkiCertificateHandler_NoRevocationTime(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/pki/revoke", func(w http.ResponseWriter, r *http.Request) {
		// Vault response with no revocation_time_rfc3339 field
		jsonResponse(w, map[string]interface{}{
			"data": map[string]interface{}{},
		})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "revoke_pki_certificate",
			Arguments: map[string]interface{}{
				"mount":         "pki",
				"serial_number": "40:1d:4a:bb",
			},
		},
	}

	result, err := revokePkiCertificateHandler(ctx, req, logger)
	require.NoError(t, err)
	assert.False(t, result.IsError, "expected success, got: %s", getResultText(result))
	assert.Contains(t, getResultText(result), "Successfully revoked certificate")
	// No "Revoked at:" suffix when revocation_time_rfc3339 is absent
	assert.NotContains(t, getResultText(result), "Revoked at:")
}
