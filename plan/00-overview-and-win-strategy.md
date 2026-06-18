# 00 — Overview & Win Strategy

## One-sentence pitch

> We extended **Vault MCP** with a **Vault Transit** capability suite, giving **Bob** a new
> security capability for guiding and executing encryption-as-a-service workflows directly
> through Vault.

## Problem statement

Modern applications constantly need to encrypt sensitive data, but managing encryption keys
directly introduces operational and security risk: key storage, rotation, access control,
auditability, and cryptographic correctness are all easy to get wrong.

HashiCorp Vault **Transit** solves this with *encryption-as-a-service* — applications send
data to Vault for cryptographic operations while Vault holds and manages the keys. But using
Transit still requires understanding Vault paths, payload formats, base64 requirements, key
versions, and specific API behavior.

**The gap we close:** developers and security teams need a simpler, guided, *agentic* way to
perform Vault Transit workflows through Vault MCP without hand-crafting low-level Vault API
calls.

## Solution

Add a new `pkg/tools/transit/` capability area to the existing `hashicorp/vault-mcp-server`
so Bob can:

```
Create key → Read metadata → Encrypt → Decrypt → Rotate key
           → Rewrap old ciphertext to latest version → Generate/verify HMAC
```

This is not a single function — it's a **coherent capability suite** spanning key management,
encryption operations, key lifecycle, and data-integrity verification, plus a reusable
**agentic-asset suite** that turns the work into a replicable pattern.

## Two ways we use Bob

1. **As an SDLC assistant** — planning, design, implementation, testing, documentation, demo prep.
2. **As the agentic interface** — invoking the new Transit MCP tools to perform real Vault crypto.

This "Bob builds it, then Bob uses it" loop is the core narrative of the submission.

---

## Judging rubric → our strategy

The Bob-a-thon scores three things. We engineer the project to hit all three deliberately.

### 1. Use of Bob across the SDLC
> *How broadly did the team use Bob, beyond coding (design, testing, docs, ops)?*

- Every workstream records the prompts it used in `docs/bob-usage-log.md`, tagged by SDLC phase.
- Bob is used for: capability selection, tool/API design, Go implementation, unit + e2e test
  generation, documentation, demo-script authoring, and operational runbooks.
- The demo explicitly shows Bob **operating** the tools, not just having written them.

See [07-demo-and-bob-usage.md](07-demo-and-bob-usage.md).

### 2. Complexity of enhancement
> *Preference for enhancing/reimagining complex existing systems; deep understanding; transformation > greenfield.*

- We extend a **real, non-trivial production MCP server**, not a toy. We adopt its exact
  conventions: session-scoped Vault clients, MCP tool annotations, mount detection, structured
  logging, error semantics, and test harness.
- We demonstrate deep understanding by shipping an **"add a new Vault engine to vault-mcp-server"
  playbook** ([05-agentic-assets.md](05-agentic-assets.md)) derived from doing it ourselves.
- Optional reach: extend the existing `create_mount` tool to support `transit`, and open an
  upstream PR — both signal genuine mastery of the host system.

See [01-system-architecture.md](01-system-architecture.md).

### 3. Agentic best practices (reusable / replicable)
> *What agentic best practices did the team create or formalize? Shared in a reusable format? Documented, adoptable?*

This is the **primary differentiator** and where we invest the most. We ship:

- `.github/copilot-instructions.md` — repo-wide conventions for extending vault-mcp-server.
- `.github/instructions/transit-tools.instructions.md` — path-scoped rules for the transit package.
- A `.github/prompts/` library — repeatable prompts for adding a tool, writing tests, running the demo.
- `.github/agents/vault-transit.agent.md` — a **custom Bob mode** that acts as a guarded
  security-workflow interface with a restricted toolset.
- `.github/skills/vault-transit/SKILL.md` — packaged domain knowledge.
- A documented, transferable **engine playbook** + agentic guardrails (validate-before-execute,
  safe defaults, tool annotations).

See [05-agentic-assets.md](05-agentic-assets.md).

---

## Why Vault Transit (capability selection rationale)

- **Security-relevant:** solves a real cryptographic key-management problem.
- **Demo-friendly:** the full lifecycle is easy to show end-to-end in 5 minutes.
- **Self-contained:** only requires Vault — no external cloud dependencies.
- **Technically meaningful:** key management, encryption, decryption, rotation, rewrapping,
  HMACs, and error handling.
- **Reusable:** the tool structure becomes a template for future Vault engines.

## Final framing (use this wording in the submission)

> Our project extends Vault MCP with a complete Vault Transit capability suite. With Bob as the
> agentic interface, users can create and inspect encryption keys, encrypt and decrypt data,
> rotate keys, rewrap ciphertext to newer key versions, and verify data integrity using HMACs.
> This turns Bob from a passive assistant into an active security-workflow interface for
> Vault-backed encryption-as-a-service.

## Scope decisions (locked)

| Decision | Choice |
|----------|--------|
| Base codebase | Fork of upstream `hashicorp/vault-mcp-server` (Go) |
| Tool scope | 8 core + select stretch (`list_transit_keys`, `sign_data`, `verify_signature`, `hash_data`, `generate_random_bytes`) |
| Transit mount | Default `transit`; enabled via CLI in setup (optional `create_mount` extension as stretch) |
| Agentic assets | Full reusable suite (key differentiator) |
| Demo client | Bob in VS Code (`.vscode/mcp.json`); MCP Inspector fallback |
| Parallelism | 5 workstreams; orchestrator owns shared files + integration |
