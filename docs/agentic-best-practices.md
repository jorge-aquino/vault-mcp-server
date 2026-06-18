# Agentic Best Practices

> Six principles we formalized while building the Vault Transit capability suite for
> `hashicorp/vault-mcp-server`. Each principle has a concrete implementation example from this
> project and a transferable rule for future work.

---

## 1. Tool-bounded actions

**Principle:** Bob invokes structured MCP tools; it never manages raw cryptographic material or
Vault state directly.

**Why it matters:** An agent with unrestricted access to a Vault API can perform arbitrary
operations including key export, mount deletion, or policy changes. Scoping the agent's
capability to a defined tool set limits the blast radius of mistakes and makes the agent's
behavior auditable.

**Implementation in this project:**
The `vault-transit.agent.md` custom mode gives Bob exactly 9 tools: the core Transit lifecycle
plus `list_transit_keys`. It does not include `delete_transit_key`, `export_transit_key`, or
any KV or PKI tools. If a user asks Bob to export a key while in the Vault Transit Security
mode, Bob cannot do it — the capability simply is not available.

```yaml
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
```

**Transfer rule:** Define your agent's toolset around the minimum set of capabilities needed for
its intended workflows. Use separate agent files for different security zones (e.g. one agent
for read-only audit workflows, one for write operations). Document what the agent cannot do.

---

## 2. Clear tool contracts

**Principle:** Every tool has defined parameters, return values, known Vault behavior, and
honest tool annotations. Bob can reason about tool behavior before invoking it.

**Why it matters:** Ambiguous tools lead to over-calling (invoking a tool to probe its behavior)
and under-trusting (not invoking a tool because the outcome is unclear). Clear contracts make
agents faster and more reliable.

**Implementation in this project:**
Each tool in `plan/04-tool-specifications.md` has an explicit contract table:
- Required vs optional parameters with types and defaults
- The exact Vault API path invoked
- The response fields returned and their types
- Tool annotations that honestly describe behavior (`ReadOnlyHint`, `DestructiveHint`, `IdempotentHint`)

For example, `read_transit_key` is annotated `ReadOnlyHint=true, IdempotentHint=true` — Bob
knows it is safe to call multiple times without side effects. `rotate_transit_key` is annotated
`IdempotentHint=false` — Bob knows it changes state and should confirm before calling.

The return value of `decrypt_data` returns both the raw base64 plaintext *and* the decoded
UTF-8 text, so Bob can present the result meaningfully without a second tool call.

**Transfer rule:** Write tool descriptions *for an LLM to read*, not for human developers. Be
explicit about what the tool does, what it doesn't do, and what state it changes. Set
`DestructiveHint=true` on any operation that cannot be undone.

---

## 3. Safe defaults

**Principle:** When parameters are optional, default to the most secure option. The agent should
produce safe outcomes without the user having to know the security implications of each choice.

**Why it matters:** Most developers do not know that Transit keys can be made exportable, or
that plaintext backups are an option. Without safe defaults, an agent following user intent
("create me a Transit key") could accidentally create an insecure key configuration.

**Implementation in this project:**
In `create_transit_key`:
- `exportable` defaults to `false` — key material cannot leave Vault
- `allow_plaintext_backup` defaults to `false` — no unencrypted key export
- `derived` defaults to `false` — simpler operation for the common case
- `type` defaults to `aes256-gcm96` — authenticated encryption, widely supported

In `transit_helpers.go`:
- `extractBool(args, "exportable", false)` — the default is enforced in code, not just documentation

In the `vault-transit.agent.md` persona:
> "Always create keys as non-exportable. Do not set exportable=true unless explicitly requested
> and the user understands the risk."

**Transfer rule:** For every optional boolean that has a "safe" vs "convenient" option, default
to safe in code *and* in the agent persona. Document the reason for the default so future
maintainers don't change it casually.

---

## 4. Validate before execute

**Principle:** Inputs are validated before any Vault call is made. The agent receives a clear,
actionable error message before reaching the network boundary.

**Why it matters:** Calling Vault with invalid input is wasteful (unnecessary network round-trip),
produces low-quality error messages (Vault errors are not always user-friendly), and can have
unexpected side effects (some Vault calls are non-idempotent even on error).

**Implementation in this project:**
In `transit_helpers.go`, every validation concern is a standalone function:

```go
validateKeyName(name)      // rejects empty, "/", ".."
validateBase64(s)          // rejects invalid standard base64
validateCiphertext(ct)     // rejects strings without "vault:v" prefix
```

In every handler, these run before `client.GetVaultClientFromContext`:

```go
if err := validateKeyName(name); err != nil {
    return mcp.NewToolResultError(err.Error()), nil  // nil Go error — tool-level failure
}
// ...
if err := validateCiphertext(ciphertext); err != nil {
    return mcp.NewToolResultError(err.Error()), nil
}
// only now do we call Vault
vault, err := client.GetVaultClientFromContext(ctx, logger)
```

The unit tests verify this explicitly: `decrypt_data_test.go` registers the mock endpoint with
a `handlerCalled` flag and asserts the flag is `false` when a malformed ciphertext is passed.
The validation fires before the Vault client is even obtained.

**Transfer rule:** Establish a validation layer for every external call your tools make. Test
that bad input is rejected *before* the external call. Make error messages actionable — state
what was wrong and how to fix it.

---

## 5. Human-readable feedback

**Principle:** Success and error messages are written for the human (or agent) reading them, not
as raw technical output.

**Why it matters:** An agent that surfaces raw Vault errors (`error decoding response: unexpected
end of JSON input`) leaves the human without guidance. An agent that translates errors into
actionable plain English ("The key 'my-key' does not exist — run create_transit_key first")
enables the user to self-recover.

**Implementation in this project:**
Tool success messages include context:
```
"Created Transit key 'customer-data' (type=aes256-gcm96, exportable=false) in mount 'transit'.
 You can now encrypt data with encrypt_data."
```

HMAC verification returns a verdict:
```
"HMAC verified successfully: the input matches the HMAC."
"HMAC verification failed: the input does not match the provided HMAC."
```

The `vault-transit.agent.md` persona explicitly translates Vault errors:
```
"The ciphertext format is invalid — it should start with vault:v."
"Vault returned 403 — check that your token has transit/encrypt/* permissions."
```

**Transfer rule:** Write success messages that tell the user what happened and what to do next.
Write error messages that name the problem, give the expected value or format, and suggest a
fix. Never surface a raw `%v` error string as the only output.

---

## 6. Reusable patterns

**Principle:** The Transit package, its agentic assets, and its documentation are designed to
be a template — not a one-off. Any future Vault secrets engine should be able to follow the
same pattern.

**Why it matters:** Agentic assets have compounding value. The second engine you add benefits
from the first's playbook, prompt library, and shared conventions. Without deliberate
codification, each addition requires rediscovering the same patterns.

**Implementation in this project:**

The pattern is explicit and documented:

| Asset | Purpose | Reusable as |
|-------|---------|------------|
| `pkg/tools/transit/` | Package structure | Template for `pkg/tools/ssh/`, `pkg/tools/database/`, etc. |
| `transit_helpers.go` | Shared validation and extraction | Model for `<engine>_helpers.go` |
| `transit_test.go` | Shared test infrastructure | Model for `<engine>_test.go` |
| `.github/instructions/transit-tools.instructions.md` | Path-scoped rules | Template with `applyTo: "pkg/tools/<engine>/**"` |
| `.github/prompts/add-transit-tool.prompt.md` | Scaffolding prompt | Adapt for any engine |
| `.github/prompts/write-tool-tests.prompt.md` | Test generation prompt | Reusable as-is |
| `docs/add-a-new-vault-engine.md` | Step-by-step playbook | The replication guide |

The playbook (`docs/add-a-new-vault-engine.md`) documents the 8-step process from scoping to
agentic layer, with the complete `create_transit_key.go` as a worked example. A developer
adding SSH engine support can follow it without having been part of the original team.

**Transfer rule:** Before finalizing any capability suite, ask: "Could a developer who was not
part of this project build the next one from these artifacts alone?" If not, add what's missing.
Document the *why* behind decisions, not just the *what*.
