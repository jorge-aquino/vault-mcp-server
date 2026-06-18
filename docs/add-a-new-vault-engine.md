# Add a New Vault Secrets Engine to vault-mcp-server

> A replicable, transferable playbook for extending `hashicorp/vault-mcp-server` with any new
> Vault secrets engine. Built from first principles while implementing the Transit capability
> suite. A developer who was not part of the original team can follow this guide to add SSH,
> Database, LDAP, or any other Vault engine.

---

## Overview

The vault-mcp-server uses a consistent pattern for all capabilities: one Go package per engine,
one file per tool, shared helpers, central registration in `tools.go`, and a parallel agentic
asset layer that makes the tools discoverable and safely usable through Bob.

```
pkg/tools/
├── kv/           # Key-Value (existing, use as a secondary reference)
├── pki/          # PKI certificates (existing, mount-based pattern)
├── sys/          # Mount management
├── transit/      # Transit encryption-as-a-service (the worked example)
└── tools.go      # Central registration — THE ONLY SHARED EDIT POINT
```

Adding a new engine means: scaffold the package → implement tools → register → test → document → add agentic layer.

---

## Step 1 — Decide scope

Before writing any code, define the scope explicitly. Underdefined scope leads to tools that are
too broad (one tool does five things) or too narrow (five tools that all do one thing poorly).

### Questions to answer

1. **Which engine?** Name the Vault secrets engine and its canonical mount path.
   Example: `transit` → mounted at `transit`; `database` → mounted at `database`.

2. **Which endpoints?** Map the API surface you want to expose. List every endpoint you intend
   to wrap as a tool. Use the [Vault API docs](https://developer.hashicorp.com/vault/api-docs)
   as your authoritative source.

3. **Core vs stretch?** Which tools are essential for a coherent demo? Which are nice-to-have?
   Ship core first. Example: for Transit, `create/read/rotate/encrypt/decrypt/rewrap/hmac/verify`
   are core; `sign/verify_signature/hash/random` are stretch.

4. **What are the safe defaults?** What do unsafe optional parameters look like? Document them
   before implementation so they are enforced in code, not just docs.

### Scope template

```
Engine: <name>
Mount: <default mount path>
Core tools (ship first):
  - <tool_name>: <HTTP method> <path> → <returns>
  - ...
Stretch tools:
  - <tool_name>: <HTTP method> <path> → <returns>
Safe defaults:
  - <param>: default to <safe value> because <reason>
```

---

## Step 2 — Scaffold the package

Create the package directory and two shared files before writing any tool:

```bash
mkdir -p pkg/tools/<engine>
```

### 2a. Create `<engine>_helpers.go`

This file holds shared functions used by every tool in the package. Never duplicate these
across tool files. Model it after
[`pkg/tools/transit/transit_helpers.go`](../pkg/tools/transit/transit_helpers.go).

**Minimum contents:**

```go
// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package <engine>

import (
    "fmt"
    "strings"

    "github.com/hashicorp/vault/api"
)

const DefaultMount = "<engine>"

// <engine>Path builds the Vault API path for this engine.
func <engine>Path(mount, segment, name string) string {
    mount = strings.Trim(mount, "/")
    if name == "" {
        return fmt.Sprintf("%s/%s", mount, segment)
    }
    return fmt.Sprintf("%s/%s/%s", mount, segment, name)
}

// resolveMount returns the mount or DefaultMount when empty.
func resolveMount(args map[string]interface{}) string {
    if m, ok := args["mount"].(string); ok && strings.TrimSpace(m) != "" {
        return strings.Trim(m, "/")
    }
    return DefaultMount
}

// extractString, extractBool, extractInt — same as transit_helpers.go
// validateKeyName — validate non-empty, no "/" or ".."
// dataString — safe field extraction from *api.Secret

// Add engine-specific validators here, e.g.:
// func validateAlgorithm(alg string, allowed []string) error { ... }
```

Standard helpers to include (copy from `transit_helpers.go`, adapt as needed):
- `resolveMount(args) string`
- `<engine>Path(mount, segment, name) string`
- `extractString(args, key, required) (string, error)`
- `extractBool(args, key, def) bool`
- `extractInt(args, key, def) int`
- `validateKeyName(name) error` (or equivalent resource-name validator)
- `dataString(secret *api.Secret, key) (string, error)`

### 2b. Create `<engine>_test.go`

This file holds shared test helpers used by every `*_test.go` in the package. Must be in the
same package (not `_test` suffix) so tool tests can access it.

```go
// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package <engine>

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

type fakeSession struct {
    id      string
    notifCh chan mcp.JSONRPCNotification
}

func (f fakeSession) Initialize()                                         {}
func (f fakeSession) Initialized() bool                                   { return true }
func (f fakeSession) NotificationChannel() chan<- mcp.JSONRPCNotification { return f.notifCh }
func (f fakeSession) SessionID() string                                   { return f.id }

func newLogger() *log.Logger {
    logger := log.New()
    logger.SetLevel(log.ErrorLevel)
    return logger
}

func newTestContext(t *testing.T, mux *http.ServeMux) (context.Context, func()) {
    t.Helper()
    mockVault := httptest.NewServer(mux)
    sessionID := "test-<engine>-" + t.Name()
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

func jsonResponse(w http.ResponseWriter, body interface{}) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(body) //nolint:errcheck
}

func getResultText(r *mcp.CallToolResult) string {
    if r == nil || len(r.Content) == 0 { return "" }
    tc, ok := mcp.AsTextContent(r.Content[0])
    if !ok { return "" }
    return tc.Text
}

func decodeBody(r *http.Request, v interface{}) {
    json.NewDecoder(r.Body).Decode(v) //nolint:errcheck
}
```

---

## Step 3 — One file per tool

Create one `.go` file per tool, following the constructor + handler pattern exactly.

### Tool file structure

```
pkg/tools/<engine>/
├── <engine>_helpers.go       # shared helpers (Step 2a)
├── <engine>_test.go          # shared test helpers (Step 2b)
├── <tool_one>.go             # constructor + handler
├── <tool_one>_test.go        # unit tests
├── <tool_two>.go
├── <tool_two>_test.go
└── ...
```

### Constructor + handler pattern

Every tool follows this exact structure. Do not deviate.

```go
// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package <engine>

import (
    "context"
    "fmt"

    "github.com/hashicorp/vault-mcp-server/pkg/client"
    "github.com/hashicorp/vault-mcp-server/pkg/utils"
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
    log "github.com/sirupsen/logrus"
)

// ToolName creates a tool for doing X.
func ToolName(logger *log.Logger) server.ServerTool {
    return server.ServerTool{
        Tool: mcp.NewTool("tool_name",
            mcp.WithToolAnnotation(mcp.ToolAnnotation{
                ReadOnlyHint:    utils.ToBoolPtr(false), // true for read/list/verify
                DestructiveHint: utils.ToBoolPtr(false), // true only if irreversible
                IdempotentHint:  utils.ToBoolPtr(true),  // false for rotate/create-unique
            }),
            mcp.WithDescription("What this tool does. Written for an LLM to understand and act on."),
            mcp.WithString("mount",
                mcp.Description("<Engine> mount path. Defaults to '<engine>'.")),
            mcp.WithString("name",
                mcp.Required(),
                mcp.Description("Name of the resource.")),
            // ... additional params ...
        ),
        Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            return toolNameHandler(ctx, req, logger)
        },
    }
}

func toolNameHandler(ctx context.Context, req mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
    logger.Debug("Handling tool_name request")

    // 1. Assert arguments type
    args, ok := req.Params.Arguments.(map[string]interface{})
    if !ok {
        return mcp.NewToolResultError("Missing or invalid arguments format"), nil
    }

    // 2. Resolve mount (always optional, always first)
    mount := resolveMount(args)

    // 3. Extract and validate all required params BEFORE calling Vault
    name, err := extractString(args, "name", true)
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }
    if err := validateKeyName(name); err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }
    // ... other validations ...

    // 4. Get the session-scoped Vault client
    vault, err := client.GetVaultClientFromContext(ctx, logger)
    if err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("Failed to get Vault client: %v", err)), nil
    }

    // 5. Build payload and call Vault
    payload := map[string]interface{}{
        "field": value,
    }
    secret, err := vault.Logical().Write(<engine>Path(mount, "segment", name), payload)
    if err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("Failed to X '%s': %v", name, err)), nil
    }

    // 6. Extract response fields
    result, err := dataString(secret, "field")
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }

    // 7. Return human-readable success message
    logger.WithFields(log.Fields{"mount": mount, "name": name}).Info("X succeeded")
    return mcp.NewToolResultText(fmt.Sprintf("Result: %s. Next step: ...", result)), nil
}
```

### Which Vault API method to use

| Operation | Vault Go SDK call | Notes |
|-----------|-------------------|-------|
| Create/update | `vault.Logical().Write(path, payload)` | SDK sends as HTTP PUT |
| Read | `vault.Logical().Read(path)` | HTTP GET |
| List | `vault.Logical().List(path)` | HTTP LIST |
| Delete | `vault.Logical().Delete(path)` | HTTP DELETE |
| System ops | `vault.Sys().ListMounts()` etc. | Sys API |

### Handling nil secrets

`vault.Logical().Write()` returns `(secret *api.Secret, err error)`. For operations that
create resources with no response body (e.g. `POST transit/keys/:name`), Vault returns a
`204 No Content` and `secret` is `nil` with `err == nil`. This is success — do not treat
nil as an error for write operations that do not return data.

For read operations, a nil secret with nil error means the resource does not exist (404):

```go
secret, err := vault.Logical().Read(path)
if err != nil {
    return mcp.NewToolResultError(...), nil
}
if secret == nil {
    return mcp.NewToolResultError(fmt.Sprintf("'%s' not found", name)), nil
}
```

### Worked example — `create_transit_key.go`

This is the canonical reference implementation. Study it before writing your first tool.

```go
// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package transit

import (
    "context"
    "fmt"

    "github.com/hashicorp/vault-mcp-server/pkg/client"
    "github.com/hashicorp/vault-mcp-server/pkg/utils"
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
    log "github.com/sirupsen/logrus"
)

func CreateTransitKey(logger *log.Logger) server.ServerTool {
    return server.ServerTool{
        Tool: mcp.NewTool("create_transit_key",
            mcp.WithToolAnnotation(mcp.ToolAnnotation{
                ReadOnlyHint:    utils.ToBoolPtr(false),
                DestructiveHint: utils.ToBoolPtr(false),
                IdempotentHint:  utils.ToBoolPtr(true),
            }),
            mcp.WithDescription(
                "Create a new named encryption key in the Vault Transit secrets engine. "+
                    "Keys are created non-exportable by default. Use this before encrypting data."),
            mcp.WithString("mount",
                mcp.Description("Transit mount path. Defaults to 'transit'.")),
            mcp.WithString("name",
                mcp.Required(),
                mcp.Description("Name of the encryption key to create, e.g. 'customer-data'.")),
            mcp.WithString("type",
                mcp.Description("Key type. Defaults to 'aes256-gcm96'.")),
            mcp.WithBoolean("exportable",
                mcp.Description("Allow key export. Defaults to false (recommended).")),
            mcp.WithBoolean("allow_plaintext_backup",
                mcp.Description("Allow plaintext backup. Defaults to false.")),
        ),
        Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            return createTransitKeyHandler(ctx, req, logger)
        },
    }
}
```

**Key design decisions visible here:**
- `ReadOnlyHint=false` — this creates state
- `DestructiveHint=false` — key creation is reversible (key can be deleted)
- `IdempotentHint=true` — calling create twice with the same name is safe (Vault ignores it)
- `mount` is optional with an inline description that states the default
- `name` is `Required()` with a concrete example in the description
- `exportable` defaults to `false` — the safe default is enforced by `extractBool(args, "exportable", false)`

---

## Step 4 — Register in `tools.go`

`pkg/tools/tools.go` is the single shared registration point. It is **orchestrator-owned** —
tool authors report their constructor names; the orchestrator appends registrations.

Add one `AddTool` pair per tool, grouped by engine with a comment:

```go
import (
    // ... existing imports ...
    "<module>/pkg/tools/<engine>"
)

func InitTools(hcServer *server.MCPServer, logger *log.Logger) {
    // ... existing registrations ...

    // Tools for <Engine> secrets engine
    toolOne := <engine>.ToolOne(logger)
    hcServer.AddTool(toolOne.Tool, toolOne.Handler)

    toolTwo := <engine>.ToolTwo(logger)
    hcServer.AddTool(toolTwo.Tool, toolTwo.Handler)
    // ...
}
```

**Do not register until the tool's unit tests pass.** Registration before tests means a broken
tool is live in the server.

---

## Step 5 — Test

### Unit tests (required for every tool)

See `docs/testing-strategy.md` for the full pattern. Minimum per tool:
1. Happy path — assert the Vault request payload (not just the response)
2. Missing required parameter
3. Engine-specific validation failures (bad format, unsupported value)
4. Vault error (mock returns 4xx/5xx)

Run and verify:
```bash
gofmt -l ./pkg/tools/<engine>/   # must print nothing
go vet ./pkg/tools/<engine>/...  # must be clean
go test ./pkg/tools/<engine>/... # must pass
```

### e2e test (required for lifecycle coverage)

Create `e2e/<engine>_e2e_test.go` with build tag `//go:build e2e`. Cover the full lifecycle:
create resource → operate → verify state → clean up.

See `e2e/transit_e2e_test.go` for the structure.

### Regression

After registration, run the full test suite to confirm no regressions:
```bash
make test
```

---

## Step 6 — Document

### README section

Add a `## <Engine> Tools` section to `README.md` following the Transit Tools section as a
template. Include:
- Two-sentence explanation of what the engine does
- Prerequisites (if any beyond vault server -dev)
- Setup commands (enable the engine)
- Tool list table with brief descriptions
- Quick-start example Bob prompt

### Tool examples doc

Create `docs/<engine>-tool-examples.md` with one runnable example per tool. See
`docs/transit-tool-examples.md` as the reference.

### Troubleshooting section

In the tool examples doc, include a section covering common errors:
- Engine not enabled (`vault secrets enable <engine>`)
- Token permission issues (what policy capabilities are needed)
- Engine-specific format requirements

---

## Step 7 — Add the agentic layer

The agentic layer is what makes the capability suite reusable and teachable. It has four parts.

### 7a. Path-scoped instructions

Create `.github/instructions/<engine>-tools.instructions.md`:

```yaml
---
applyTo: "pkg/tools/<engine>/**"
---
# <Engine> tool authoring rules

- Use shared helpers from `<engine>_helpers.go`: [list them]
- `mount` is optional, defaults to `<engine>`
- [engine-specific conventions]
- Constructor and tool names must match the spec exactly: [table]
- Each tool has a table-driven `_test.go`
- Never edit `tools.go` or `<engine>_helpers.go`
```

### 7b. Scaffolding prompt

Create `.github/prompts/add-<engine>-tool.prompt.md` following the structure of
`.github/prompts/add-transit-tool.prompt.md`:

```yaml
---
mode: agent
description: Scaffold a new <Engine> MCP tool end to end.
---
# Add a new <Engine> tool
...
Do not edit tools.go or <engine>_helpers.go.
```

### 7c. Custom agent mode (optional but valuable)

Create `.github/agents/<engine>.agent.md` if the engine warrants a dedicated persona:

```yaml
---
name: Vault <Engine>
description: Agentic interface for <Engine> workflows.
tools:
  - <tool_one>
  - <tool_two>
---
# Guarded <Engine> assistant
...
```

### 7d. Skill file (optional)

Create `.github/skills/vault-<engine>/SKILL.md` with the mental model, canonical workflow,
tool cheat-sheet, and gotchas. See `.github/skills/vault-transit/SKILL.md` as the template.

---

## Step 8 — Guardrails checklist

Before marking the engine as complete, verify all of these:

### Safe defaults
- [ ] All optional parameters with security implications default to the safe value
- [ ] Safe defaults are enforced in code (not just documented)
- [ ] `exportable`, `allow_plaintext_backup`, and similar dangerous options require explicit opt-in
- [ ] Key creation uses the most widely supported algorithm type by default

### Validate before execute
- [ ] Every required parameter is validated before calling Vault
- [ ] String parameters have non-empty/non-whitespace checks
- [ ] Format-specific inputs (base64, ciphertext prefixes, paths) have dedicated validators
- [ ] Validation runs before `client.GetVaultClientFromContext` — no Vault call on bad input
- [ ] Unit tests verify that bad input does NOT reach the mock Vault endpoint

### No secret logging
- [ ] No `logger.Info/Debug/Error` calls include plaintext values, tokens, or key material
- [ ] Log statements use field names only: `log.Fields{"mount": mount, "name": name}`
- [ ] Success messages do not include sensitive response fields (only safe identifiers)

### Honest tool annotations
- [ ] `ReadOnlyHint=true` only for operations that truly change no state
- [ ] `DestructiveHint=true` for any operation that cannot be undone (delete, revoke)
- [ ] `IdempotentHint=false` for operations that change state on every call (rotate, increment)

### Documentation completeness
- [ ] README section added
- [ ] Tool examples doc created
- [ ] Troubleshooting section covers the 3 most common errors
- [ ] `bob-usage-log.md` has entries for this engine's implementation

### Agentic layer
- [ ] Path-scoped instructions file created
- [ ] Scaffolding prompt created
- [ ] Agent mode created (if warranted)
- [ ] Skill file created (if applicable)

---

## Quick reference — files created per engine

```
pkg/tools/<engine>/
├── <engine>_helpers.go
├── <engine>_test.go
├── <tool_one>.go + <tool_one>_test.go
├── <tool_two>.go + <tool_two>_test.go
└── ...

e2e/<engine>_e2e_test.go

docs/<engine>-tool-examples.md

.github/instructions/<engine>-tools.instructions.md
.github/prompts/add-<engine>-tool.prompt.md
.github/agents/<engine>.agent.md          (optional)
.github/skills/vault-<engine>/SKILL.md    (optional)
```

**Single shared edit (orchestrator only):**
```
pkg/tools/tools.go   — AddTool registrations
README.md            — new engine section
```

---

## Transfer checklist — applying this playbook to a new engine

1. [ ] Scope defined: engine name, endpoints, core/stretch split, safe defaults
2. [ ] `pkg/tools/<engine>/` created with helpers + test helper files
3. [ ] At least one core tool implemented, tested (`go test` passes), and registered
4. [ ] All core tools implemented and passing `make test`
5. [ ] e2e lifecycle test written
6. [ ] README section added
7. [ ] Tool examples doc created
8. [ ] Path-scoped instructions file created
9. [ ] Scaffolding prompt created
10. [ ] Guardrails checklist signed off

## Engines to consider next

| Engine | Default mount | Interesting capability | Complexity |
|--------|-------------|----------------------|-----------|
| SSH | `ssh` | Signed certificates for host/client access | Medium |
| Database | `database` | Dynamic database credentials | Medium-High |
| TOTP | `totp` | MFA / two-factor code generation | Low |
| LDAP | `ldap` | Dynamic LDAP credentials | Medium |
| Transform | `transform` | Format-preserving encryption (FPE) | Medium |
| PKI (advanced) | `pki` | Certificate issuance beyond what's already there | Low (incremental) |
