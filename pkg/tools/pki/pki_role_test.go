// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package pki

// Tests for createPkiRoleHandler, readPkiRoleHandler, and deletePkiRoleHandler.
// Shared test helpers (fakeSession, newLogger, newTestContext, jsonResponse,
// getResultText) live in pki_test.go.
// pkiMountsResponse lives in pki_certificate_test.go.

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// createPkiRoleHandler
// ---------------------------------------------------------------------------

func TestCreatePkiRoleHandler_Success(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, pkiMountsResponse("pki"))
	})
	mux.HandleFunc("/v1/pki/roles/web-server", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		jsonResponse(w, map[string]interface{}{})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "create_pki_role",
		Arguments: map[string]interface{}{
			"mount":     "pki",
			"role_name": "web-server",
			"max_ttl":   "30d",
		},
	}}

	result, err := createPkiRoleHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError, "expected success: %s", getResultText(result))
	assert.Contains(t, getResultText(result), "web-server")
}

// TestCreatePkiRoleHandler_BoolsOmitted is the regression test for the former
// bare .(bool) panic bug. allow_any_name and allow_glob_domains are entirely
// absent from the args map; the handler must apply their defaults without
// panicking.
func TestCreatePkiRoleHandler_BoolsOmitted(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, pkiMountsResponse("pki"))
	})
	mux.HandleFunc("/v1/pki/roles/minimal-role", func(w http.ResponseWriter, r *http.Request) {
		// Verify the defaults were applied in the written payload.
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
		assert.Equal(t, true, body["allow_any_name"], "allow_any_name should default to true")
		assert.Equal(t, false, body["allow_glob_domains"], "allow_glob_domains should default to false")
		jsonResponse(w, map[string]interface{}{})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	// Neither allow_any_name nor allow_glob_domains present in args.
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "create_pki_role",
		Arguments: map[string]interface{}{
			"mount":     "pki",
			"role_name": "minimal-role",
		},
	}}

	result, err := createPkiRoleHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError, "expected success with omitted bools: %s", getResultText(result))
}

// TestCreatePkiRoleHandler_BoolsExplicit verifies that explicitly supplied
// boolean values are forwarded verbatim to Vault.
func TestCreatePkiRoleHandler_BoolsExplicit(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, pkiMountsResponse("pki"))
	})
	mux.HandleFunc("/v1/pki/roles/glob-role", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
		assert.Equal(t, false, body["allow_any_name"])
		assert.Equal(t, true, body["allow_glob_domains"])
		jsonResponse(w, map[string]interface{}{})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "create_pki_role",
		Arguments: map[string]interface{}{
			"mount":              "pki",
			"role_name":          "glob-role",
			"allow_any_name":     false,
			"allow_glob_domains": true,
		},
	}}

	result, err := createPkiRoleHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError, "expected success with explicit bools: %s", getResultText(result))
}

func TestCreatePkiRoleHandler_MissingMount(t *testing.T) {
	logger := newLogger()

	ctx, cleanup := newTestContext(t, http.NewServeMux())
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "create_pki_role",
		Arguments: map[string]interface{}{
			"role_name": "web-server",
		},
	}}

	result, err := createPkiRoleHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "mount")
}

func TestCreatePkiRoleHandler_MissingRoleName(t *testing.T) {
	logger := newLogger()

	ctx, cleanup := newTestContext(t, http.NewServeMux())
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "create_pki_role",
		Arguments: map[string]interface{}{
			"mount": "pki",
		},
	}}

	result, err := createPkiRoleHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "role_name")
}

func TestCreatePkiRoleHandler_MountNotFound(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		// "pki" is not in the mounts map.
		jsonResponse(w, pkiMountsResponse("other-mount"))
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "create_pki_role",
		Arguments: map[string]interface{}{
			"mount":     "pki",
			"role_name": "web-server",
		},
	}}

	result, err := createPkiRoleHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "pki")
}

func TestCreatePkiRoleHandler_VaultWriteError(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, pkiMountsResponse("pki"))
	})
	mux.HandleFunc("/v1/pki/roles/broken-role", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "create_pki_role",
		Arguments: map[string]interface{}{
			"mount":     "pki",
			"role_name": "broken-role",
		},
	}}

	result, err := createPkiRoleHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}

// ---------------------------------------------------------------------------
// readPkiRoleHandler
// ---------------------------------------------------------------------------

func TestReadPkiRoleHandler_Success(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, pkiMountsResponse("pki"))
	})
	mux.HandleFunc("/v1/pki/roles/web-server", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		jsonResponse(w, map[string]interface{}{
			"data": map[string]interface{}{
				"allow_any_name": true,
				"max_ttl":        "720h",
			},
		})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "read_pki_role",
		Arguments: map[string]interface{}{
			"mount":     "pki",
			"role_name": "web-server",
		},
	}}

	result, err := readPkiRoleHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError, "expected success: %s", getResultText(result))
	text := getResultText(result)
	assert.Contains(t, text, "allow_any_name")
	assert.Contains(t, text, "720h")
}

func TestReadPkiRoleHandler_MissingMount(t *testing.T) {
	logger := newLogger()

	ctx, cleanup := newTestContext(t, http.NewServeMux())
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "read_pki_role",
		Arguments: map[string]interface{}{
			"role_name": "web-server",
		},
	}}

	result, err := readPkiRoleHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "mount")
}

func TestReadPkiRoleHandler_MissingRoleName(t *testing.T) {
	logger := newLogger()

	ctx, cleanup := newTestContext(t, http.NewServeMux())
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "read_pki_role",
		Arguments: map[string]interface{}{
			"mount": "pki",
		},
	}}

	result, err := readPkiRoleHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "role_name")
}

func TestReadPkiRoleHandler_MountNotFound(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, pkiMountsResponse("other-mount"))
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "read_pki_role",
		Arguments: map[string]interface{}{
			"mount":     "pki",
			"role_name": "web-server",
		},
	}}

	result, err := readPkiRoleHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "pki")
}

// TestReadPkiRoleHandler_RoleNotFound covers the nil-secret branch: Vault
// returns 404 for a non-existent role, which the SDK translates to a nil
// secret with no error.
func TestReadPkiRoleHandler_RoleNotFound(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, pkiMountsResponse("pki"))
	})
	mux.HandleFunc("/v1/pki/roles/ghost", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "read_pki_role",
		Arguments: map[string]interface{}{
			"mount":     "pki",
			"role_name": "ghost",
		},
	}}

	result, err := readPkiRoleHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "ghost")
}

func TestReadPkiRoleHandler_VaultError(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, pkiMountsResponse("pki"))
	})
	mux.HandleFunc("/v1/pki/roles/broken-role", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "read_pki_role",
		Arguments: map[string]interface{}{
			"mount":     "pki",
			"role_name": "broken-role",
		},
	}}

	result, err := readPkiRoleHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}

// ---------------------------------------------------------------------------
// deletePkiRoleHandler
// ---------------------------------------------------------------------------

func TestDeletePkiRoleHandler_Success(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, pkiMountsResponse("pki"))
	})
	mux.HandleFunc("/v1/pki/roles/old-role", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusNoContent)
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "delete_pki_role",
		Arguments: map[string]interface{}{
			"mount":     "pki",
			"role_name": "old-role",
		},
	}}

	result, err := deletePkiRoleHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError, "expected success: %s", getResultText(result))
	assert.Contains(t, getResultText(result), "old-role")
}

func TestDeletePkiRoleHandler_MissingMount(t *testing.T) {
	logger := newLogger()

	ctx, cleanup := newTestContext(t, http.NewServeMux())
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "delete_pki_role",
		Arguments: map[string]interface{}{
			"role_name": "old-role",
		},
	}}

	result, err := deletePkiRoleHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "mount")
}

func TestDeletePkiRoleHandler_MissingRoleName(t *testing.T) {
	logger := newLogger()

	ctx, cleanup := newTestContext(t, http.NewServeMux())
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "delete_pki_role",
		Arguments: map[string]interface{}{
			"mount": "pki",
		},
	}}

	result, err := deletePkiRoleHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "role_name")
}

func TestDeletePkiRoleHandler_MountNotFound(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, pkiMountsResponse("other-mount"))
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "delete_pki_role",
		Arguments: map[string]interface{}{
			"mount":     "pki",
			"role_name": "old-role",
		},
	}}

	result, err := deletePkiRoleHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "pki")
}

func TestDeletePkiRoleHandler_VaultError(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, pkiMountsResponse("pki"))
	})
	mux.HandleFunc("/v1/pki/roles/broken-role", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "delete_pki_role",
		Arguments: map[string]interface{}{
			"mount":     "pki",
			"role_name": "broken-role",
		},
	}}

	result, err := deletePkiRoleHandler(ctx, req, logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}
