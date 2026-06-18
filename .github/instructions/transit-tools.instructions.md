---
applyTo: "pkg/tools/transit/**"
---

# Transit tool authoring rules

## Shared helpers — use these; never duplicate them

All helpers live in [`transit_helpers.go`](../../pkg/tools/transit/transit_helpers.go). Use them
exclusively:

| Helper | Signature | Purpose |
|--------|-----------|---------|
| `resolveMount` | `(args map[string]interface{}) string` | Returns `args["mount"]` or `"transit"` |
| `transitPath` | `(mount, segment, name string) string` | Builds `mount/segment/name` Vault path |
| `extractString` | `(args, key, required) (string, error)` | Typed extraction with required-check |
| `extractBool` | `(args, key, def) bool` | Typed bool extraction with default |
| `extractInt` | `(args, key, def) int` | Typed int extraction (handles float64 from JSON) |
| `validateKeyName` | `(name string) error` | Rejects empty, `/`, and `..` in names |
| `validateBase64` | `(s string) error` | Rejects invalid standard base64 |
| `validateCiphertext` | `(ct string) error` | Rejects strings missing `vault:v` prefix |
| `dataString` | `(secret *api.Secret, key string) (string, error)` | Safe typed field extraction from Vault response |

## Parameter conventions

- `mount` is **optional** on every tool. Default to `transit` via `resolveMount(args)`.
- Plaintext and HMAC `input` parameters are **base64 at the Vault API boundary**.
  - Accept raw text from the user by default (auto-encode with `base64.StdEncoding.EncodeToString`).
  - When `*_is_base64=true`, call `validateBase64` first, then pass as-is.
- Always call `validateCiphertext(ct)` before any `decrypt_data` or `rewrap_data` Vault call.

## Tool annotations

| Group | ReadOnly | Destructive | Idempotent |
|-------|----------|-------------|------------|
| read, list, verify, hash, random | `true` | `false` | `true` |
| create, rotate, encrypt, decrypt, rewrap, hmac, sign | `false` | `false` | varies |

No Transit operation hard-deletes keys, so `DestructiveHint` is always `false`.

## Constructor and tool name contract

Names must match `plan/04-tool-specifications.md` exactly:

| Constructor | Tool name |
|-------------|-----------|
| `CreateTransitKey` | `create_transit_key` |
| `ReadTransitKey` | `read_transit_key` |
| `RotateTransitKey` | `rotate_transit_key` |
| `ListTransitKeys` | `list_transit_keys` |
| `EncryptData` | `encrypt_data` |
| `DecryptData` | `decrypt_data` |
| `RewrapData` | `rewrap_data` |
| `GenerateHMAC` | `generate_hmac` |
| `VerifyHMAC` | `verify_hmac` |
| `SignData` | `sign_data` |
| `VerifySignature` | `verify_signature` |
| `HashData` | `hash_data` |
| `GenerateRandomBytes` | `generate_random_bytes` |

## Testing requirements

Each tool must have a `_test.go` with table-driven tests using the package helpers
(`newLogger`, `newTestContext`, `jsonResponse`, `getResultText`, `decodeBody`) from
[`transit_test.go`](../../pkg/tools/transit/transit_test.go):

- **Happy path**: assert the Vault request payload and the success result text.
- **Missing required param**: result must have `IsError == true`.
- **Bad base64** (where applicable): rejected before any Vault call.
- **Malformed ciphertext** (decrypt/rewrap): rejected pre-call on missing `vault:v` prefix.
- **Vault error**: mock returns 4xx/5xx; handler returns `NewToolResultError`.

## Off-limits files

**Never edit `tools.go` or `transit_helpers.go`.** These are orchestrator-owned. Report your
constructor name so the orchestrator can register it.
