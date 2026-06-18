# 03 — Foundation Setup (Phase 0)

> Owned by the **orchestrator**. This is the blocking prerequisite for WS-A/B/C. Keep it small
> and fast: get a building fork, a running Vault with Transit enabled, Bob wired up, and the
> shared `transit_helpers.go` + `transit_test.go` scaffolding in place.

## 0.1 Prerequisites

- Go **1.24+** (`go version`)
- Docker (for Vault dev container, optional if using the binary)
- `git`, `make`
- HashiCorp Vault CLI (`vault`) — for enabling the Transit engine and e2e
- Node (for `npx @modelcontextprotocol/inspector`, optional)
- VS Code with Bob

## 0.2 Fork & clone

```bash
# Fork hashicorp/vault-mcp-server to your org, then:
git clone https://github.com/<your-org>/vault-mcp-server.git
cd vault-mcp-server
git remote add upstream https://github.com/hashicorp/vault-mcp-server.git
git checkout -b feat/transit

make build           # confirm a clean baseline build
make test            # confirm existing tests pass before we change anything
```

## 0.3 Run Vault (dev) + enable Transit

```bash
# Terminal 1: dev server (root token printed to stdout; dev mode is in-memory, NOT for prod)
vault server -dev -dev-root-token-id="root"

# Terminal 2: point CLI/SDK at it and enable Transit
export VAULT_ADDR="http://127.0.0.1:8200"
export VAULT_TOKEN="root"
vault secrets enable transit
vault write -f transit/keys/demo-key        # smoke test the engine
```

> **Docker alternative**
> ```bash
> docker network create mcp
> docker run --cap-add=IPC_LOCK --name=vault-dev --network=mcp -p 8200:8200 \
>   hashicorp/vault server -dev
> docker logs vault-dev   # grab the root token
> ```

### Optional enhancement (stretch, shows system mastery)
The existing `create_mount` tool supports `kv/kv2/pki`. Optionally extend it to accept
`transit` so Bob can enable the engine itself. If not done, enabling via CLI in setup is fine
and documented in the demo script.

## 0.4 Wire Bob in VS Code (`.vscode/mcp.json`)

Create `.vscode/mcp.json` in the repo for the **stdio** transport (simplest for the demo):

```jsonc
{
  "inputs": [
    { "type": "promptString", "id": "vault_token", "description": "Vault Token", "password": true }
  ],
  "servers": {
    "vault-mcp-server": {
      "command": "${workspaceFolder}/vault-mcp-server",
      "args": ["stdio"],
      "env": {
        "VAULT_ADDR": "http://127.0.0.1:8200",
        "VAULT_TOKEN": "${input:vault_token}"
      }
    }
  }
}
```

HTTP-mode alternative (if running `./vault-mcp-server http --transport-port 8080`):

```jsonc
{
  "inputs": [
    { "type": "promptString", "id": "vault_token", "description": "Vault Token", "password": true }
  ],
  "servers": {
    "vault-mcp-server": {
      "url": "http://localhost:8080/mcp?VAULT_ADDR=http://127.0.0.1:8200",
      "headers": { "X-Vault-Token": "${input:vault_token}" }
    }
  }
}
```

Reload VS Code; confirm Bob lists the existing Vault tools before adding Transit.

## 0.5 Create the transit package scaffolding

Create the directory and two **orchestrator-owned** shared files. These unblock all tool
workstreams.

```
pkg/tools/transit/
├── transit_helpers.go     # shared, orchestrator-owned
└── transit_test.go        # shared test helpers, orchestrator-owned
```

### `transit_helpers.go` — required surface

Implement these so subagents can rely on them (signatures are the contract):

```go
// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package transit

import (
    "encoding/base64"
    "fmt"
    "strings"

    "github.com/hashicorp/vault/api"
)

// DefaultMount is the conventional Transit mount path.
const DefaultMount = "transit"

// transitPath builds a Vault Transit API path, e.g. transitPath("transit","keys","demo")
// -> "transit/keys/demo". segment is one of: keys, encrypt, decrypt, rewrap, hmac, verify,
// sign, hash, random. When name is empty, the trailing slash/name is omitted.
func transitPath(mount, segment, name string) string {
    mount = strings.Trim(mount, "/")
    if name == "" {
        return fmt.Sprintf("%s/%s", mount, segment)
    }
    return fmt.Sprintf("%s/%s/%s", mount, segment, name)
}

// resolveMount returns the provided mount or DefaultMount when empty.
func resolveMount(args map[string]interface{}) string { /* ... */ }

// extractString returns a string arg; errors if required and missing/empty.
func extractString(args map[string]interface{}, key string, required bool) (string, error) { /* ... */ }

// extractBool / extractInt: optional typed extraction with defaults.
func extractBool(args map[string]interface{}, key string, def bool) bool { /* ... */ }
func extractInt(args map[string]interface{}, key string, def int) int { /* ... */ }

// validateKeyName guards against empty/whitespace/path-traversal in key names.
func validateKeyName(name string) error { /* ... */ }

// validateBase64 ensures s is valid standard base64.
func validateBase64(s string) error {
    if _, err := base64.StdEncoding.DecodeString(s); err != nil {
        return fmt.Errorf("value is not valid base64: %w", err)
    }
    return nil
}

// validateCiphertext ensures Vault ciphertext format: must start with "vault:v".
func validateCiphertext(ct string) error {
    if !strings.HasPrefix(ct, "vault:v") {
        return fmt.Errorf("invalid ciphertext: expected a 'vault:v<version>:...' value")
    }
    return nil
}

// dataString safely pulls a string field out of a *api.Secret response.
func dataString(secret *api.Secret, key string) (string, error) { /* ... */ }
```

> Keep helper bodies small and well-tested. They are the dependency surface for A/B/C.

### `transit_test.go` — shared test helpers

Port the helpers used by the KV/PKI tests into the transit package (they are package-private):

```go
// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package transit

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/mark3labs/mcp-go/mcp"
    log "github.com/sirupsen/logrus"
)

func newLogger() *log.Logger { /* logrus at Debug, discard or testing writer */ }

// newTestContext starts an httptest server with mux, builds an MCP session context whose
// Vault client points at it, and returns (ctx, cleanup).
func newTestContext(t *testing.T, mux *http.ServeMux) (context.Context, func()) { /* ... */ }

func jsonResponse(w http.ResponseWriter, body interface{}) { /* set header + json.Encode */ }

func getResultText(r *mcp.CallToolResult) string { /* extract text content for assertions */ }
```

> Mirror how `pkg/tools/kv/*_test.go` constructs its context and binds a session Vault client
> to the mock server. If the existing helpers live in `kv` only, replicate the minimal pieces
> here; do not import test code across packages.

## 0.6 Registration stub in `tools.go`

Add the import and an (initially empty) Transit block. The orchestrator fills it as tools land:

```go
import (
    // ...
    "github.com/hashicorp/vault-mcp-server/pkg/tools/transit"
)

func InitTools(hcServer *server.MCPServer, logger *log.Logger) {
    // ... existing registrations ...

    // Tools for Transit encryption-as-a-service
    // (registrations appended as transit tools are delivered by WS-A/B/C)
}
```

## 0.7 Phase 0 exit criteria (all must pass)

- [ ] Fork builds: `make build` succeeds on `feat/transit`.
- [ ] Baseline tests pass: `make test` green before new tools.
- [ ] Vault dev server reachable; `transit/` engine enabled; `transit/keys/demo-key` exists.
- [ ] `.vscode/mcp.json` created; Bob lists existing Vault tools.
- [ ] `pkg/tools/transit/transit_helpers.go` compiles and exports the contract surface.
- [ ] `pkg/tools/transit/transit_test.go` compiles; a trivial helper test passes.
- [ ] `transit` import + empty registration block present in `tools.go` (compiles).

Once green, dispatch WS-A/B/C/E and (already running) WS-D.
