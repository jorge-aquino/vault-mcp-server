---
mode: agent
description: Drive the live Transit demo workflow through the vault-mcp-server tools.
---

# Vault Transit demo runbook

You are running a live demonstration of the Vault Transit capability suite. Execute each step
using the vault-mcp-server MCP tools, narrate what happened and **why it matters**, then move
to the next step.

## Pre-flight check

Before starting, verify:
1. Vault dev server is running: `vault status` should return `Sealed: false`
2. Transit engine is enabled: `vault secrets list` should include `transit/`
3. If transit is not enabled, run: `vault secrets enable transit`
4. The vault-mcp-server is running and connected

If any check fails, stop and fix it before proceeding.

---

## Step 1 — Create the key

**Tool:** `create_transit_key`
**Arguments:** `{ "name": "customer-data" }`

Call the tool now. After it succeeds, explain:
- What was created (a named AES-256-GCM96 key in Vault's Transit engine)
- Why the key is non-exportable by default (Vault holds the key material; apps never touch it)
- Why this matters (no key sprawl, no accidental exposure in logs or environment variables)

---

## Step 2 — Read the key metadata

**Tool:** `read_transit_key`
**Arguments:** `{ "name": "customer-data" }`

Call the tool. After it returns, explain:
- The current version is `1`, meaning no rotations have happened yet
- The `supports_encryption` flag shows this key can encrypt data
- Reading metadata is safe and read-only — no state changes

---

## Step 3 — Encrypt data

**Tool:** `encrypt_data`
**Arguments:** `{ "name": "customer-data", "plaintext": "the quick brown fox" }`

Call the tool. After it returns:
- Show the returned ciphertext (`vault:v1:...`)
- Explain the `vault:v1:` prefix encodes the key version — critical for the rotation story
- Note that the plaintext never left the Vault boundary unencrypted
- **Save the ciphertext** — you will use it in Steps 4, 6, and 7

---

## Step 4 — Decrypt the ciphertext

**Tool:** `decrypt_data`
**Arguments:** `{ "name": "customer-data", "ciphertext": "<ciphertext from Step 3>" }`

Call the tool. After it returns:
- Confirm the original plaintext was recovered ("the quick brown fox")
- This proves the round-trip works
- Explain that Vault decodes the base64 and returns the original text

---

## Step 5 — Rotate the key

**Tool:** `rotate_transit_key`
**Arguments:** `{ "name": "customer-data" }`

Call the tool. After it returns:
- Explain that a new key version (v2) was created
- The old v1 ciphertext is still decryptable — Vault keeps key history
- **New encryptions will use v2** automatically
- This is non-destructive: no existing ciphertext breaks

---

## Step 6 — Rewrap the old ciphertext

**Tool:** `rewrap_data`
**Arguments:** `{ "name": "customer-data", "ciphertext": "<ciphertext from Step 3>" }`

Call the tool. After it returns:
- Show that the new ciphertext has `vault:v2:` prefix
- Explain why this matters: after deprecating v1, old ciphertext can be upgraded
- **Key point:** rewrap never exposes the plaintext — Vault decrypts and re-encrypts internally
- This is why you use `rewrap_data` instead of decrypt + encrypt

---

## Step 7 — Generate an HMAC

**Tool:** `generate_hmac`
**Arguments:** `{ "name": "customer-data", "input": "the quick brown fox" }`

Call the tool. After it returns:
- Show the `vault:v...` HMAC value
- Explain HMACs prove data integrity and authenticity (not just confidentiality)
- The same key used for encryption can also generate HMACs

---

## Step 8 — Verify the HMAC

**Tool:** `verify_hmac`
**Arguments:** `{ "name": "customer-data", "input": "the quick brown fox", "hmac": "<hmac from Step 7>" }`

Call the tool. After it returns `valid: true`, also run it with a tampered input:

**Tool:** `verify_hmac`
**Arguments:** `{ "name": "customer-data", "input": "the quick brown Fox", "hmac": "<hmac from Step 7>" }`

After the tampered call returns `valid: false`, explain:
- Even a single character change invalidates the HMAC
- This detects data tampering and corruption
- Bob validated inputs and called Vault — no manual base64, no raw API calls

---

## Closing narration

After all 8 steps, summarize:
- What the workflow accomplished (full key lifecycle: create → encrypt → decrypt → rotate → rewrap → HMAC → verify)
- How Bob acted as an active security-workflow interface, not a passive assistant
- The final state: key at version 2, ciphertext upgraded to v2, data integrity verified
