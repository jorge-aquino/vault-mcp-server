---
name: vault-transit
description: Domain knowledge for Vault Transit encryption-as-a-service via vault-mcp-server MCP tools. Use when creating/rotating Transit keys, encrypting/decrypting/rewrapping data, or generating/verifying HMACs and signatures.
---

# Vault Transit skill

## When to use this skill

Load this skill for any request involving:
- Creating, reading, rotating, or listing Vault Transit encryption keys
- Encrypting or decrypting data using a Transit key
- Rewrapping ciphertext after a key rotation
- Generating or verifying HMACs or signatures via Transit
- Diagnosing Vault Transit errors (wrong key type, bad ciphertext format, base64 issues)

## Mental model

**Keys live in Vault. Key material never leaves Vault.**

The fundamental guarantee of Vault Transit is that applications never touch raw key bytes. They
send data *to* Vault for cryptographic operations, and Vault returns the result. Key rotation,
access control, auditing, and cryptographic correctness are all managed by Vault — not by the
application.

From the application's perspective, a Transit key is just a name. `customer-data` is the key.
Vault handles everything else.

## Ciphertext format

Vault Transit ciphertext always follows this structure:

```
vault:v<key_version>:<base64_encoded_ciphertext>
```

Examples:
- `vault:v1:8SDd3WHDOjf7mq69CyCqYjBk5S/OtGWf...` — encrypted with key version 1
- `vault:v2:dUI9iegXADNT...` — the same data rewrapped to key version 2 after rotation

**Always validate the `vault:v` prefix** before calling decrypt or rewrap. If the string does
not start with `vault:v`, it is not valid Transit ciphertext.

## Base64 at the Vault boundary

Vault Transit expects plaintext and HMAC inputs in **base64 encoding**. The vault-mcp-server
tools handle this automatically:

- If `plaintext_is_base64=false` (the default), the tool base64-encodes raw text before sending.
- If `plaintext_is_base64=true`, the tool validates the base64 and passes it through unchanged.
- Decrypted plaintext is returned as base64; the tools also decode it to UTF-8 when valid.

## Canonical workflow

```
1. create_transit_key(name="customer-data")
        ↓
2. read_transit_key(name="customer-data")       ← check type, version, capabilities
        ↓
3. encrypt_data(name="customer-data", plaintext="sensitive text")
        ↓  returns vault:v1:...
4. decrypt_data(name="customer-data", ciphertext="vault:v1:...")
        ↓  recovers "sensitive text"
5. rotate_transit_key(name="customer-data")     ← new key version created (v2)
        ↓
6. rewrap_data(name="customer-data", ciphertext="vault:v1:...")
        ↓  returns vault:v2:...  (no plaintext exposure)
7. generate_hmac(name="customer-data", input="data to verify")
        ↓  returns vault:v2:... hmac
8. verify_hmac(name="customer-data", input="data to verify", hmac="vault:v2:...")
        ↓  valid: true
```

## Tool cheat-sheet

| Tool | Purpose | Key params |
|------|---------|-----------|
| `create_transit_key` | Create a named key | `name` (req), `type`, `exportable`, `derived` |
| `read_transit_key` | Read key metadata and version info | `name` (req) |
| `rotate_transit_key` | Rotate key to new version | `name` (req) |
| `list_transit_keys` | List all key names | — |
| `encrypt_data` | Encrypt plaintext | `name`, `plaintext`, `plaintext_is_base64` |
| `decrypt_data` | Decrypt ciphertext | `name`, `ciphertext` |
| `rewrap_data` | Upgrade ciphertext to latest key version (no plaintext) | `name`, `ciphertext` |
| `generate_hmac` | Generate HMAC for integrity | `name`, `input`, `algorithm` |
| `verify_hmac` | Verify HMAC | `name`, `input`, `hmac` |
| `sign_data` | Sign data with asymmetric key | `name`, `input`, `hash_algorithm` |
| `verify_signature` | Verify digital signature | `name`, `input`, `signature` |
| `hash_data` | Hash data (no key needed) | `algorithm`, `input` |
| `generate_random_bytes` | Cryptographically secure random bytes | `bytes`, `format` |

All tools accept an optional `mount` parameter (default: `transit`).

## Gotchas

### Derived keys require a context on every operation
If a key was created with `derived=true`, every call to `encrypt_data`, `decrypt_data`,
`rewrap_data`, `generate_hmac`, and `verify_hmac` must include a base64-encoded `context`
parameter. The same context must be used for matching operations. Mismatched contexts produce
errors or incorrect results.

### Signing requires an asymmetric key type
`sign_data` and `verify_signature` only work with keys of types: `ed25519`, `ecdsa-p256`,
`ecdsa-p384`, `ecdsa-p521`, `rsa-2048`, `rsa-3072`, `rsa-4096`. Attempting to sign with an
`aes256-gcm96` key returns an error from Vault.

### `aes256-gcm96` is the safe default for symmetric encryption
It is authenticated (provides integrity as well as confidentiality), widely supported, and the
default when no `type` is specified. Use it unless you have a specific reason to choose
otherwise.

### Rewrap, not decrypt + encrypt
When upgrading ciphertext to a newer key version, always use `rewrap_data`. It performs
decrypt-and-re-encrypt entirely within Vault without ever surfacing the plaintext over the API.
Using `decrypt_data` followed by `encrypt_data` is less safe (plaintext crosses the API boundary
twice) and more complex.

### Key rotation is non-destructive
After rotation, all existing ciphertext remains decryptable. Vault keeps the key history.
New encryptions automatically use the latest version. Use `read_transit_key` to see the current
`latest_version` and `min_decryption_version`.

## Error reference

| Error pattern | Likely cause | Fix |
|--------------|-------------|-----|
| `invalid ciphertext: expected a 'vault:v<version>:...' value` | Input doesn't start with `vault:v` | Check ciphertext source; ensure it came from a Transit encrypt call |
| `value is not valid base64` | Raw binary or truncated base64 passed | Ensure padding (`=`); use standard encoding |
| `key name must not contain '/' or '..'` | Path traversal attempt in key name | Use simple alphanumeric names with hyphens |
| `missing required parameter 'name'` | Tool called without key name | Add `name` parameter |
| Vault 403 | Token lacks permission | Ensure token has `transit/encrypt/*` capability |
| Vault 404 on key | Key does not exist | Run `create_transit_key` first |
