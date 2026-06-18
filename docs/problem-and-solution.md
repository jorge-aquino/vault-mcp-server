# Problem and Solution

## The Problem: Encryption Key Management Is Hard to Get Right

Modern applications constantly need to encrypt sensitive data — customer records, PII,
authentication tokens, financial information. But managing encryption keys directly creates a
cascade of operational and security risk that is easy to underestimate.

When an application manages its own keys, it must solve: where to store the key securely, how to
rotate it without downtime, how to audit who accessed it, how to ensure the cryptographic
implementation is correct, how to handle key versioning, and how to revoke access. In practice,
keys end up in environment variables, hardcoded in configs, or stored in KV databases with
insufficient access control. They get copied between environments. They never get rotated.

HashiCorp Vault's **Transit secrets engine** solves this through *encryption-as-a-service*.
Applications send data to Vault for cryptographic operations while Vault holds and manages the
keys. The plaintext key material never leaves Vault. Key rotation, auditing, versioning, and
access control are Vault's job.

But Transit itself still has a usability gap. Using it correctly requires understanding Vault API
paths, base64 encoding requirements, ciphertext version formats (`vault:v<N>:...`), key type
selection, and the difference between `decrypt+encrypt` versus the safer `rewrap` operation. A
developer who does not know these details will use Transit incorrectly or avoid it entirely.

## The Solution: Vault Transit Tools + Bob as the Agentic Interface

This project extends the `hashicorp/vault-mcp-server` with a complete **Vault Transit capability
suite**: 13 structured MCP tools covering key management, encryption operations, key lifecycle,
and data integrity verification.

The tools give Bob — as an AI agent operating through the Model Context Protocol — a typed,
validated interface to Vault Transit. Bob does not hand-craft Vault API calls. It invokes
structured tools with defined parameters, validated inputs, and human-readable outputs.

**Bob isn't just explaining Transit — it's executing real Vault crypto.**

## The Workflow

The canonical workflow the tools enable:

```
create_transit_key("customer-data")
        ↓
read_transit_key     → verify type, version, capabilities
        ↓
encrypt_data         → "the quick brown fox" → vault:v1:...
        ↓
decrypt_data         → vault:v1:... → "the quick brown fox"  (round-trip verified)
        ↓
rotate_transit_key   → key now at version 2; existing ciphertext still works
        ↓
rewrap_data          → vault:v1:... → vault:v2:...  (no plaintext exposed)
        ↓
generate_hmac        → "the quick brown fox" → vault:v2:... (HMAC)
        ↓
verify_hmac          → valid: true  /  tampered input → valid: false
```

Each step in this workflow is a single Bob invocation. Bob validates the inputs (base64, key
name, ciphertext format), calls Vault through the MCP tool, and explains the result in plain
language. A developer who has never used Vault Transit before can execute this full lifecycle
with Bob guiding and executing each step.

## Why This Matters

There are three layers of value here.

**For application developers:** The Transit tools eliminate the need to understand Vault API
internals. Bob handles base64 encoding, ciphertext version parsing, and error translation.
Developers can use production-grade encryption without becoming Vault experts.

**For security teams:** The agentic guardrails built into the tools enforce safe defaults that
are often skipped when using Vault directly. Keys are created non-exportable by default. Inputs
are validated before any Vault call. Tool annotations signal intent — read-only operations are
marked as such, and nothing in this suite is destructive.

**For engineering teams:** The Transit package is a replicable pattern. The same structure —
shared helpers, one file per tool, table-driven tests, path-scoped Copilot instructions,
scaffolding prompts — can be applied to any Vault secrets engine. The
`docs/add-a-new-vault-engine.md` playbook codifies exactly how we built this, so the next
engine (SSH, Database, LDAP) follows the same path.

The "Bob builds it, then Bob uses it" loop is the core narrative of this project. Bob assisted
across the entire SDLC — planning, design, implementation, testing, documentation, and demo
preparation — and then serves as the runtime interface for the tools it helped create.
