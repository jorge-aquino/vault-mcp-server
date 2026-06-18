// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package transit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/vault-mcp-server/pkg/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

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
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	return logger
}

// newTestContext starts an httptest server with mux, builds an MCP session context whose
// Vault client points at it, and returns (ctx, cleanup).
func newTestContext(t *testing.T, mux *http.ServeMux) (context.Context, func()) {
	t.Helper()
	mockVault := httptest.NewServer(mux)

	sessionID := "test-transit-" + t.Name()
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

// getResultText extracts text content from a CallToolResult for test assertions.
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

// decodeBody is a tiny helper wrapping json.NewDecoder(r.Body).Decode(&v).
func decodeBody(r *http.Request, v interface{}) {
	json.NewDecoder(r.Body).Decode(v) //nolint:errcheck
}
