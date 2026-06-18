# 01 — System Architecture & Code Patterns

> Required reading for **every subagent**. This captures how `hashicorp/vault-mcp-server`
> actually works so new Transit tools match existing conventions exactly. Demonstrating this
> understanding is worth real points under the "complexity of enhancement" rubric.

## 1. Repository overview

`hashicorp/vault-mcp-server` is a Go implementation of a Model Context Protocol (MCP) server
that integrates with HashiCorp Vault. It supports both **stdio** and **StreamableHTTP**
transports, making it compatible with Bob/Claude Desktop/VS Code and other MCP clients.

- **Language:** Go (`.go-version` targets Go 1.24/1.25 toolchain).
- **MCP library:** `github.com/mark3labs/mcp-go` (`mcp` + `server` packages).
- **Vault SDK:** `github.com/hashicorp/vault/api`.
- **Logging:** `github.com/sirupsen/logrus`.
- **Tests:** `github.com/stretchr/testify` (`assert`, `require`) + `net/http/httptest`.
- **License header (every file):**
  ```go
  // Copyright IBM Corp. 2025, 2026
  // SPDX-License-Identifier: MPL-2.0
  ```

### Project structure (existing)

```
vault-mcp-server/
├── cmd/vault-mcp-server/        # main entrypoint (init.go, main.go)
├── pkg/
│   ├── client/                  # Vault client + HTTP middleware
│   │   ├── client.go            # session-scoped Vault client management
│   │   └── middleware.go        # CORS, logging, Vault context
│   ├── tools/                   # MCP tools
│   │   ├── kv/                  # Key-Value tools (template we follow)
│   │   ├── pki/                 # PKI tools (mount-based, closest analogue)
│   │   ├── sys/                 # mount management (create/list/delete mount)
│   │   └── tools.go             # InitTools(): central tool registration
│   └── utils/                   # ExtractMountPath, ToBoolPtr, etc.
├── e2e/                         # end-to-end tests
├── Makefile                     # build/test automation
├── Dockerfile
└── go.mod
```

**We add:** `pkg/tools/transit/` (new package) and register it in `pkg/tools/tools.go`.

## 2. Tool definition pattern (replicate exactly)

Each tool is a constructor function returning a `server.ServerTool` with a `Tool` (schema) and
a `Handler` (logic). Real example from `pkg/tools/kv/write_secret.go`, abbreviated:

```go
package kv

import (
    "context"
    "fmt"

    "github.com/hashicorp/vault-mcp-server/pkg/client"
    "github.com/hashicorp/vault-mcp-server/pkg/utils"
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
    log "github.com/sirupsen/logrus"
)

// WriteSecret creates a tool for writing secrets to a Vault KV mount.
func WriteSecret(logger *log.Logger) server.ServerTool {
    return server.ServerTool{
        Tool: mcp.NewTool("write_secret",
            mcp.WithToolAnnotation(mcp.ToolAnnotation{
                DestructiveHint: utils.ToBoolPtr(true),
                IdempotentHint:  utils.ToBoolPtr(false),
            }),
            mcp.WithDescription("Writes a secret value to a KV store..."),
            mcp.WithString("mount", mcp.Required(), mcp.Description("...")),
            mcp.WithString("path", mcp.Required(), mcp.Description("...")),
            mcp.WithString("key", mcp.Required(), mcp.Description("...")),
            mcp.WithString("value", mcp.Required(), mcp.Description("...")),
        ),
        Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            return writeSecretHandler(ctx, req, logger)
        },
    }
}
```

### Handler pattern

```go
func writeSecretHandler(ctx context.Context, req mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
    // 1. Extract & assert args
    args, ok := req.Params.Arguments.(map[string]interface{})
    if !ok {
        return mcp.NewToolResultError("Missing or invalid arguments format"), nil
    }

    // 2. Validate inputs (return NewToolResultError on bad input — note: error is nil)
    mount, err := utils.ExtractMountPath(args)
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }

    // 3. Get the session's Vault client
    vault, err := client.GetVaultClientFromContext(ctx, logger)
    if err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("Failed to get Vault client: %v", err)), nil
    }

    // 4. Call Vault via Logical()/Sys()
    res, err := vault.Logical().Write(fullPath, payload)
    if err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("Failed to write: %v", err)), nil
    }

    // 5. Return human-readable success text
    return mcp.NewToolResultText(successMsg), nil
}
```

**Conventions to honor:**
- Bad input / Vault failures return `mcp.NewToolResultError(msg)` with a **`nil` Go error**
  (the error channel is reserved for protocol failures, not tool-level errors).
- Success returns `mcp.NewToolResultText(...)`.
- Use structured `logger.WithFields(...).Debug/Info/Error(...)`.
- Tool/param names are `snake_case`; descriptions are written *for an LLM to read*.

## 3. Vault client access

```go
vault, err := client.GetVaultClientFromContext(ctx, logger) // -> *vault/api.Client
```

- Clients are **session-scoped** and cached in a `sync.Map` keyed by MCP session id.
- Config comes from context (HTTP headers/query) or env: `VAULT_ADDR`, `VAULT_TOKEN`,
  `VAULT_NAMESPACE`, `VAULT_SKIP_VERIFY`.
- Vault calls use the generic logical API — **no Transit-specific SDK needed**:
  - `vault.Logical().Write(path, map[string]interface{}{...})`
  - `vault.Logical().Read(path)`
  - `vault.Logical().List(path)` (for `LIST` endpoints)
  - `vault.Sys().ListMounts()`

A returned `*api.Secret` exposes `secret.Data map[string]interface{}`.

## 4. Tool registration

All tools are wired in `pkg/tools/tools.go`:

```go
func InitTools(hcServer *server.MCPServer, logger *log.Logger) {
    // ... existing kv / pki / sys tools ...

    // Tools for Transit encryption-as-a-service  <-- WE ADD THIS BLOCK
    createKey := transit.CreateTransitKey(logger)
    hcServer.AddTool(createKey.Tool, createKey.Handler)
    // ... one AddTool pair per transit tool ...
}
```

> **Conflict-avoidance:** `tools.go` is the single shared edit point. The **orchestrator owns
> it** and adds registrations as tool files land, so subagents never touch it.

## 5. Test pattern (mock Vault over HTTP)

Tests spin up an `httptest` server that mimics Vault, build an MCP context bound to it, then
call the handler directly. Shared helpers (`newLogger`, `newTestContext`, `jsonResponse`,
`getResultText`, mount-response builders) live in a package-level `*_test.go`.

```go
func TestWriteSecretHandler_ExistingSecretV2(t *testing.T) {
    logger := newLogger()
    mux := http.NewServeMux()
    mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
        jsonResponse(w, mountsV2Response("secrets"))
    })
    mux.HandleFunc("/v1/secrets/data/app/config", func(w http.ResponseWriter, r *http.Request) {
        // switch on r.Method, return JSON
    })

    ctx, cleanup := newTestContext(t, mux)
    defer cleanup()

    req := mcp.CallToolRequest{Params: mcp.CallToolParams{
        Name: "write_secret",
        Arguments: map[string]interface{}{"mount": "secrets", "path": "app/config", "key": "k", "value": "v"},
    }}

    result, err := writeSecretHandler(ctx, req, logger)
    require.NoError(t, err)
    assert.False(t, result.IsError, "expected success: %s", getResultText(result))
}
```

> The transit package needs **its own copy** of these helpers in `transit_test.go` because Go
> test helpers are package-private. Phase 0 creates them.

## 6. Transit secrets engine — API map

All paths assume the engine is mounted at `transit` (configurable). Plaintext/ciphertext/HMAC
inputs are **base64-encoded** per Vault requirements.

| Tool | Method | Path | Key request fields | Response field |
|------|--------|------|--------------------|----------------|
| `create_transit_key` | POST | `transit/keys/:name` | `type`, `derived`, `exportable`, `allow_plaintext_backup`, `auto_rotate_period` | (200, no body) |
| `read_transit_key` | GET | `transit/keys/:name` | — | `type`, `keys{}`, `min_decryption_version`, `latest_version`, `supports_*` |
| `list_transit_keys` (stretch) | LIST | `transit/keys` | — | `keys[]` |
| `rotate_transit_key` | POST | `transit/keys/:name/rotate` | — | (new version) |
| `encrypt_data` | POST | `transit/encrypt/:name` | `plaintext`(b64), `context?`, `key_version?`, `nonce?` | `ciphertext` = `vault:vN:...` |
| `decrypt_data` | POST | `transit/decrypt/:name` | `ciphertext`, `context?` | `plaintext`(b64) |
| `rewrap_data` | POST | `transit/rewrap/:name` | `ciphertext`, `context?` | `ciphertext` (latest version) |
| `generate_hmac` | POST | `transit/hmac/:name/:algorithm` | `input`(b64), `key_version?` | `hmac` = `vault:vN:...` |
| `verify_hmac` | POST | `transit/verify/:name/:algorithm` | `input`(b64), `hmac` | `valid` (bool) |
| `sign_data` (stretch) | POST | `transit/sign/:name/:hash_algorithm` | `input`(b64), `signature_algorithm?` | `signature` |
| `verify_signature` (stretch) | POST | `transit/verify/:name/:hash_algorithm` | `input`(b64), `signature` | `valid` (bool) |
| `hash_data` (stretch) | POST | `transit/hash/:algorithm` | `input`(b64), `format?` | `sum` |
| `generate_random_bytes` (stretch) | POST | `transit/random/:bytes` | `format?` (hex/base64) | `random_bytes` |

### Important Transit semantics
- **Key types:** default `aes256-gcm96`. Also `aes128-gcm96`, `chacha20-poly1305`,
  `ed25519`, `ecdsa-p256/384/521`, `rsa-2048/3072/4096`, `hmac`. Signing requires an asymmetric type.
- **Ciphertext format:** `vault:v<keyversion>:<base64>`. Validate the `vault:v` prefix before
  decrypt/rewrap.
- **Rotation:** adds a new version; old ciphertext stays decryptable. `rewrap` upgrades old
  ciphertext to the latest version **without** exposing plaintext.
- **HMAC/sign algorithms:** `sha2-224/256/384/512`, `sha3-*`. Algorithm can be passed in the URL.
- **Derived keys:** if `derived=true`, every encrypt/decrypt/rewrap must include a `context` (b64).
- **Upsert:** `encrypt` can auto-create a key unless disabled; we still expose explicit
  `create_transit_key` for clarity and safe defaults.

## 7. Build, run, inspect

```bash
make build                       # -> ./vault-mcp-server (or bin/)
./vault-mcp-server               # stdio (default) — used by Bob/VS Code
./vault-mcp-server http --transport-port 8080   # StreamableHTTP

# Inspect tools interactively:
npx @modelcontextprotocol/inspector ./vault-mcp-server          # stdio
npx @modelcontextprotocol/inspector http://localhost:8080/mcp   # http
```

Environment: `VAULT_ADDR` (default `http://127.0.0.1:8200`), `VAULT_TOKEN` (required),
`VAULT_NAMESPACE` (optional), `TRANSPORT_MODE=http` to enable HTTP.
