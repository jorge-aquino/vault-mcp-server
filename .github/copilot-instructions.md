# Vault MCP Server — Copilot/Bob instructions

## Architecture
- Go MCP server using `github.com/mark3labs/mcp-go`. Tools live in `pkg/tools/<engine>/`.
- Each tool = a constructor returning `server.ServerTool` (`.Tool` schema + `.Handler`).
- Register every tool in `pkg/tools/tools.go` `InitTools` via `hcServer.AddTool`.
- Get Vault via `client.GetVaultClientFromContext(ctx, logger)`; call `vault.Logical()`/`vault.Sys()`.

## Conventions
- Tool + parameter names are snake_case. Descriptions are written for an LLM to act on.
- Tool-level failures return `mcp.NewToolResultError(msg)` with a nil Go error.
- Validate inputs before calling Vault. Prefer safe defaults (e.g. keys non-exportable).
- Add the IBM copyright + SPDX header to every file. Run `gofmt` and `go vet`.
- Every tool has a `_test.go` with httptest-mocked Vault using the package test helpers.

## Security
- Never log secrets, tokens, plaintext, or key material.
- Use tool annotations (ReadOnlyHint/DestructiveHint/IdempotentHint) honestly.

## Project structure
```
pkg/tools/
├── kv/         # Key-Value tools (template to follow)
├── pki/        # PKI certificate tools
├── sys/        # Mount management
├── transit/    # Vault Transit encryption-as-a-service (new)
└── tools.go    # Central tool registration — orchestrator-owned
```

## Tool pattern (replicate for all engines)
1. Constructor `func ToolName(logger *log.Logger) server.ServerTool`
2. Handler `func toolNameHandler(ctx, req, logger) (*mcp.CallToolResult, error)`
3. Extract + validate args → get Vault client → call Vault → return text or error
4. `mount` param is always optional; default via engine helper (e.g. `resolveMount`)
5. All Vault writes use `vault.Logical().Write(path, payload)`; reads use `.Read(path)`

## Adding a new secrets engine
See `docs/add-a-new-vault-engine.md` for the full step-by-step playbook.
See `.github/prompts/add-transit-tool.prompt.md` for the scaffolding prompt.

## Testing
- Unit tests: httptest-mocked Vault server, table-driven, in `pkg/tools/<engine>/*_test.go`
- Shared test helpers live in `<engine>_test.go` (package-private)
- `make test` runs all unit tests; `make test-transit-e2e` runs against a live vault -dev
- Assert the Vault **request payload** (not just the response) in happy-path tests
