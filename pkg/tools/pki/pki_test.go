// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package pki

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/vault-mcp-server/pkg/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Shared test helpers
// ---------------------------------------------------------------------------

// fakeSession implements server.ClientSession for testing.
type fakeSession struct {
	id      string
	notifCh chan mcp.JSONRPCNotification
}

func (f fakeSession) Initialize()                                         {}
func (f fakeSession) Initialized() bool                                   { return true }
func (f fakeSession) NotificationChannel() chan<- mcp.JSONRPCNotification { return f.notifCh }
func (f fakeSession) SessionID() string                                   { return f.id }

// newLogger returns a logrus logger at Error level to suppress noise in tests.
func newLogger() *log.Logger {
	l := log.New()
	l.SetLevel(log.ErrorLevel)
	return l
}

// newTestContext starts an httptest server with mux, builds an MCP session
// context whose Vault client points at it, and returns (ctx, cleanup).
func newTestContext(t *testing.T, mux *http.ServeMux) (context.Context, func()) {
	t.Helper()
	mockVault := httptest.NewServer(mux)

	sessionID := "test-pki-" + t.Name()
	_, err := client.NewVaultClient(sessionID, mockVault.URL, false, "test-token", "")
	require.NoError(t, err)

	mcpSrv := server.NewMCPServer("test", "1.0")
	ctx := mcpSrv.WithContext(context.Background(), fakeSession{
		id:      sessionID,
		notifCh: make(chan mcp.JSONRPCNotification, 10),
	})

	return ctx, func() {
		mockVault.Close()
		client.DeleteVaultClient(sessionID)
	}
}

// jsonResponse writes a JSON-encoded body with Content-Type application/json.
func jsonResponse(w http.ResponseWriter, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(body) //nolint:errcheck
}

// getResultText extracts the text from the first content item of a CallToolResult.
func getResultText(r *mcp.CallToolResult) string {
	if r == nil || len(r.Content) == 0 {
		return ""
	}
	tc, ok := mcp.AsTextContent(r.Content[0])
	if !ok {
		return ""
	}
	return tc.Text
}

// pkiMountsResponse returns a sys/mounts response that includes a single PKI
// mount at the given path. Used across PKI test files.
func pkiMountsResponse(mountPath string) map[string]interface{} {
	return map[string]interface{}{
		mountPath + "/": map[string]interface{}{"type": "pki"},
	}
}

// emptyMountsResponse returns a sys/mounts response with only the built-in
// sys/ mount. The Vault client's ListMounts requires secret.Data to be
// non-nil and decodable into map[string]*MountOutput.
func emptyMountsResponse() map[string]interface{} {
	return map[string]interface{}{
		"sys/": map[string]interface{}{"type": "system", "description": "system"},
	}
}

// contains is a small helper so test files do not need to import strings.
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// ---------------------------------------------------------------------------
// enablePkiHandler tests
// ---------------------------------------------------------------------------

func TestEnablePkiHandler_MissingOrInvalidArguments(t *testing.T) {
	logger := newLogger()
	ctx, cleanup := newTestContext(t, http.NewServeMux())
	defer cleanup()

	tests := []struct {
		name      string
		arguments interface{}
	}{
		{name: "missing arguments", arguments: nil},
		{name: "invalid arguments type", arguments: "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{
				Name:      "enable_pki",
				Arguments: tt.arguments,
			}}
			result, err := enablePkiHandler(ctx, req, logger)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result.IsError {
				t.Fatalf("expected error result, got success: %s", getResultText(result))
			}
		})
	}
}

func TestEnablePkiHandler_Success(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, emptyMountsResponse())
	})
	mux.HandleFunc("/v1/sys/mounts/custom-pki", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/v1/sys/mounts/custom-pki/tune", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name: "enable_pki",
		Arguments: map[string]interface{}{
			"path":        "custom-pki",
			"description": "team certificates",
			"max_ttl":     "720h",
		},
	}}

	result, err := enablePkiHandler(ctx, req, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", getResultText(result))
	}
	if text := getResultText(result); !contains(text, "custom-pki") {
		t.Fatalf("unexpected result text: %q", text)
	}
}

func TestEnablePkiHandler_DefaultPath(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, emptyMountsResponse())
	})
	mux.HandleFunc("/v1/sys/mounts/pki", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/v1/sys/mounts/pki/tune", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	// path omitted — should default to "pki" without erroring
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name:      "enable_pki",
		Arguments: map[string]interface{}{},
	}}

	result, err := enablePkiHandler(ctx, req, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success with default path, got error: %s", getResultText(result))
	}
}

func TestEnablePkiHandler_MountAlreadyExists(t *testing.T) {
	logger := newLogger()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, map[string]interface{}{
			"existing-pki/": map[string]interface{}{"type": "pki"},
		})
	})

	ctx, cleanup := newTestContext(t, mux)
	defer cleanup()

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{
		Name:      "enable_pki",
		Arguments: map[string]interface{}{"path": "existing-pki"},
	}}

	result, err := enablePkiHandler(ctx, req, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected error for existing mount")
	}
	if text := getResultText(result); !contains(text, "already exist") {
		t.Fatalf("unexpected error text: %q", text)
	}
}
