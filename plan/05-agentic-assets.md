# 05 — Agentic Assets (WS-D, the differentiator)

> The judging criterion **"Agentic best practices — created/formalized, reusable/replicable,
> documented, adoptable"** is where this submission separates from the pack. WS-D produces a
> complete, reusable Bob asset suite plus a transferable engine playbook. None of this touches
> Go code, so it runs fully in parallel from t0.

## Deliverables checklist

- [ ] `.github/copilot-instructions.md`
- [ ] `.github/instructions/transit-tools.instructions.md`
- [ ] `.github/prompts/add-transit-tool.prompt.md`
- [ ] `.github/prompts/write-tool-tests.prompt.md`
- [ ] `.github/prompts/transit-demo-runbook.prompt.md`
- [ ] `.github/agents/vault-transit.agent.md`  (custom Bob mode)
- [ ] `.github/skills/vault-transit/SKILL.md`
- [ ] `docs/problem-and-solution.md`
- [ ] `docs/bob-usage-log.md`
- [ ] `docs/agentic-best-practices.md`
- [ ] `docs/demo-script.md`
- [ ] `docs/transit-tool-examples.md`
- [ ] `docs/testing-strategy.md`
- [ ] `docs/add-a-new-vault-engine.md`  ← **the replicable playbook (high rubric value)**
- [ ] `README.md` → add "Transit Tools" section

---

## 1. `.github/copilot-instructions.md` (repo-wide)

Concise conventions Bob should always apply in this repo. Target content:

```markdown
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
```

## 2. `.github/instructions/transit-tools.instructions.md` (path-scoped)

```markdown
---
applyTo: "pkg/tools/transit/**"
---

# Transit tool authoring rules

- Use shared helpers from `transit_helpers.go`: `resolveMount`, `transitPath`, `extractString`,
  `extractBool`, `extractInt`, `validateKeyName`, `validateBase64`, `validateCiphertext`,
  `dataString`. Do not duplicate them.
- `mount` is optional and defaults to `transit`.
- Plaintext/HMAC `input` is base64 at the Vault boundary. Accept raw text from the user and
  base64-encode internally unless `*_is_base64` is set.
- Validate `vault:v` prefix on any ciphertext before decrypt/rewrap.
- Constructor and tool names must match plan/04-tool-specifications.md exactly.
- Each tool gets a table-driven `_test.go`: success + at least one validation-failure case.
- Never edit `tools.go` or `transit_helpers.go` — those are orchestrator-owned.
```

## 3. Prompt library — `.github/prompts/`

### `add-transit-tool.prompt.md`
```markdown
---
mode: agent
description: Scaffold a new Vault Transit MCP tool end to end.
---
Add a new Transit tool named `${input:toolName}` to `pkg/tools/transit/`.
1. Read plan/01-system-architecture.md and plan/04-tool-specifications.md.
2. Create `<file>.go` with a `Transit<Name>` constructor + `<name>Handler`, using shared helpers.
3. Map it to the correct Vault Transit endpoint and validate inputs before the call.
4. Add a `_test.go` with a success case and a validation-failure case (httptest-mocked Vault).
5. Run `gofmt` and `go test ./pkg/tools/transit/...`. Report the constructor name so the
   orchestrator can register it in tools.go.
Do not edit tools.go or transit_helpers.go.
```

### `write-tool-tests.prompt.md`
```markdown
---
mode: agent
description: Generate httptest-mocked unit tests for a Transit tool.
---
For `${file}`, write table-driven tests using the helpers in transit_test.go
(newLogger, newTestContext, jsonResponse, getResultText). Cover: happy path (assert the request
payload sent to Vault), invalid base64, malformed ciphertext (missing `vault:v`), missing
required params, and Vault returning an error. Keep tests hermetic — no real Vault.
```

### `transit-demo-runbook.prompt.md`
```markdown
---
mode: agent
description: Drive the live Transit demo through Bob.
---
Using the vault-mcp-server tools, perform and narrate this workflow against the dev Vault:
create_transit_key(name="customer-data") -> read_transit_key -> encrypt_data(plaintext="...")
-> decrypt_data(ciphertext=...) -> rotate_transit_key -> rewrap_data(old ciphertext)
-> generate_hmac -> verify_hmac. After each step, state the result and why it matters.
```

## 4. Custom Bob mode — `.github/agents/vault-transit.agent.md`

A guarded security-workflow persona with a restricted toolset. (Frontmatter keys may vary by
Bob version — keep `name`, `description`, `tools`.)

```markdown
---
name: Vault Transit Security
description: Agentic interface for Vault Transit encryption-as-a-service workflows.
tools:
  - create_transit_key
  - read_transit_key
  - rotate_transit_key
  - encrypt_data
  - decrypt_data
  - rewrap_data
  - generate_hmac
  - verify_hmac
  - list_transit_keys
---

You are a Vault Transit security assistant. You guide users through encryption-as-a-service
workflows and execute them via the Transit MCP tools.

Operating principles:
- Validate before acting: confirm the mount, key name, and that ciphertext looks like
  `vault:v<version>:...` before decrypt/rewrap.
- Safe defaults: create keys non-exportable; never request plaintext key material.
- Never print secrets or key material. Echo ciphertext/HMAC values, not plaintext keys.
- Explain failures in plain language (missing mount, unknown key, bad base64) and suggest a fix.
- Prefer the smallest correct tool. For "upgrade old ciphertext", use rewrap_data, not decrypt+encrypt.
- After multi-step workflows, summarize what changed (key version, ciphertext version).
```

## 5. `.github/skills/vault-transit/SKILL.md`

```markdown
---
name: vault-transit
description: Domain knowledge for Vault Transit encryption-as-a-service via the vault-mcp-server
  MCP tools. Use when creating/rotating Transit keys, encrypting/decrypting/rewrapping data, or
  generating/verifying HMACs and signatures.
---

# Vault Transit skill

## When to use
Any request involving Vault-backed encryption, key rotation, ciphertext rewrapping, or data
integrity (HMAC/signature) through the vault-mcp-server.

## Mental model
- Keys live in Vault; plaintext key material never leaves Vault.
- Ciphertext is `vault:v<version>:<base64>`. Rotation makes a new version; rewrap upgrades old
  ciphertext to the latest version without exposing plaintext.
- Inputs/outputs at the Vault boundary are base64.

## Canonical workflow
create key -> read metadata -> encrypt -> decrypt -> rotate -> rewrap -> hmac -> verify

## Tool cheat-sheet
(create/read/rotate/list)_transit_key, encrypt/decrypt/rewrap_data, generate/verify_hmac,
sign_data, verify_signature, hash_data, generate_random_bytes — see docs/transit-tool-examples.md.

## Gotchas
- Derived keys require a base64 `context` on every op.
- Signing needs an asymmetric key type (ed25519/rsa/ecdsa).
- `aes256-gcm96` is the safe default for symmetric encryption.
```

## 6. `docs/add-a-new-vault-engine.md` — the replicable playbook

This is the highest-leverage artifact for the "replicable/adoptable" rubric point. It distills
*how we extended the server* into a repeatable recipe other teams can follow for KV-v2-advanced,
SSH, Database, etc.

Outline:
1. **Decide scope** — pick the engine, list endpoints, choose core vs stretch tools.
2. **Scaffold** — `pkg/tools/<engine>/`, shared `<engine>_helpers.go` + `<engine>_test.go`.
3. **One file per tool** — constructor + handler; map params → Vault payload; validate first.
4. **Register** — append `AddTool` pairs in `tools.go`.
5. **Test** — httptest-mocked unit tests + an e2e lifecycle test vs `vault -dev`.
6. **Document** — README section, tool examples, troubleshooting.
7. **Agentic layer** — add path-scoped instructions, a prompt to scaffold the next tool, and
   (optionally) a custom mode. Use honest tool annotations.
8. **Guardrails checklist** — safe defaults, validate-before-execute, no secret logging.

Include the exact `create_transit_key.go` from plan/04 as the worked example.

## 7. Remaining docs (point to other plan files to avoid duplication)

- `docs/problem-and-solution.md` → adapt from [00-overview-and-win-strategy.md](00-overview-and-win-strategy.md).
- `docs/testing-strategy.md` → adapt from [06-testing-strategy.md](06-testing-strategy.md).
- `docs/demo-script.md` → adapt from [07-demo-and-bob-usage.md](07-demo-and-bob-usage.md).
- `docs/transit-tool-examples.md` → one runnable example per tool (Bob prompt + expected result).
- `docs/agentic-best-practices.md` → the 6 principles below, with concrete examples from our code.
- `docs/bob-usage-log.md` → living log (template in [07-demo-and-bob-usage.md](07-demo-and-bob-usage.md)).

## Agentic best practices we formalize (for `docs/agentic-best-practices.md`)

1. **Tool-bounded actions** — Bob invokes structured MCP tools; it never manages raw keys/secrets directly.
2. **Clear tool contracts** — every tool has defined params, return values, and known Vault behavior.
3. **Safe defaults** — keys non-exportable; no plaintext key exposure; least-privilege tool sets in the custom mode.
4. **Validate before execute** — mount/key-name/base64/ciphertext checks before any Vault call.
5. **Human-readable feedback** — actionable error messages (missing mount, bad ciphertext, base64 errors).
6. **Reusable patterns** — the transit package + playbook + prompts are a template for future engines.
