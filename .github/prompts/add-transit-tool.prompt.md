---
mode: agent
description: Scaffold a new Vault Transit MCP tool end to end.
---

# Add a new Vault Transit tool

You are adding a new tool to the `pkg/tools/transit/` package in the `hashicorp/vault-mcp-server`
repository. Follow every step below in order. Do not skip steps.

## Step 1 — Read the contracts

Read these files before writing any code:
- `plan/01-system-architecture.md` — the tool pattern, handler conventions, and test harness.
- `plan/04-tool-specifications.md` — the authoritative tool name, params, Vault path, and return values.
- `.github/instructions/transit-tools.instructions.md` — transit-specific rules.

## Step 2 — Create the tool file

Create `pkg/tools/transit/<file>.go` with:
1. IBM copyright + SPDX header: `// Copyright IBM Corp. 2025, 2026` / `// SPDX-License-Identifier: MPL-2.0`
2. `package transit`
3. A constructor `func <ConstructorName>(logger *log.Logger) server.ServerTool` returning a
   `server.ServerTool` with `.Tool` (schema) and `.Handler` (closure wrapping the handler func).
4. A handler `func <toolName>Handler(ctx, req, logger) (*mcp.CallToolResult, error)` that:
   - Asserts `req.Params.Arguments.(map[string]interface{})`
   - Calls `resolveMount(args)` for the mount
   - Extracts and validates all required parameters using `extractString`, `extractBool`, `extractInt`
   - Validates inputs (`validateKeyName`, `validateBase64`, `validateCiphertext`) **before** calling Vault
   - Calls `client.GetVaultClientFromContext(ctx, logger)`
   - Calls the correct Vault path via `transitPath(mount, segment, name)`
   - Returns `mcp.NewToolResultText(...)` on success or `mcp.NewToolResultError(msg)` (nil Go error) on failure

Use shared helpers from `transit_helpers.go`. Do not duplicate them.

## Step 3 — Create the test file

Create `pkg/tools/transit/<file>_test.go` with table-driven tests:
- Import `"net/http"`, `"testing"`, `"github.com/mark3labs/mcp-go/mcp"`, testify
- Use `newLogger()`, `newTestContext(t, mux)`, `jsonResponse(w, body)`, `getResultText(result)`, `decodeBody(r, &v)` from `transit_test.go`
- **Happy path**: register the mock Vault endpoint, call the handler, assert `result.IsError == false` and the response text contains expected values
- **Missing required param**: pass empty `Arguments`, assert `result.IsError == true`
- **Bad base64** (if tool accepts base64 input): pass `*_is_base64=true` with invalid base64, assert error before any Vault call
- **Malformed ciphertext** (if tool accepts ciphertext): pass a string without `vault:v` prefix
- **Vault error**: mock endpoint returns `http.StatusInternalServerError`, assert `result.IsError == true`

## Step 4 — Validate

Run:
```bash
gofmt -l ./pkg/tools/transit/
go vet ./pkg/tools/transit/...
go test ./pkg/tools/transit/...
```

All three must succeed with no output from `gofmt` and no failures.

## Step 5 — Report back

State:
- The constructor name (e.g. `transit.CreateTransitKey`)
- The tool name (e.g. `create_transit_key`)
- The files created

**Do not edit `tools.go` or `transit_helpers.go`.** The orchestrator registers the tool in `tools.go`.
