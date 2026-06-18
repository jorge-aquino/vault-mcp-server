# 04 — Tool Specifications & Code Skeletons

> The implementation contract for WS-A/B/C. Tool names, parameters, and constructor names here
> are **authoritative** — do not improvise. Each tool follows the pattern in
> [01-system-architecture.md](01-system-architecture.md).

## Conventions for all tools

- Package `transit`; one file per tool (`snake_case.go`) + matching `_test.go`.
- Constructor: `func Transit<Name>(logger *log.Logger) server.ServerTool`.
- Handler: `func <name>Handler(ctx context.Context, req mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error)`.
- Bad input / Vault errors → `mcp.NewToolResultError(msg)` with **nil** Go error.
- Success → `mcp.NewToolResultText(...)`. Prefer returning structured, useful text (include the
  ciphertext / version / validity so Bob can chain steps).
- `mount` is **optional**, defaults to `transit` via `resolveMount(args)`.
- Validate before calling Vault: key name, base64 inputs, `vault:v` ciphertext prefix.
- IBM copyright + SPDX header on every file.

### Tool annotation guidance

| Tool group | ReadOnlyHint | DestructiveHint | IdempotentHint |
|------------|--------------|-----------------|----------------|
| read/list/verify/hash/random | `true` | `false` | `true` |
| create/rotate/encrypt/decrypt/rewrap/hmac/sign | `false` | `false` | varies (`false` for rotate/encrypt) |

> Nothing here hard-deletes keys, so no tool sets `DestructiveHint=true`. (A future
> `delete_transit_key` would.)

---

## Core tools (8)

### 1. `create_transit_key`  — WS-A · `create_key.go`

| | |
|---|---|
| **Description** | Create a new named encryption key in the Vault Transit engine. |
| **Params** | `mount` (opt, default `transit`), `name` (req), `type` (opt, default `aes256-gcm96`), `exportable` (opt bool, default `false`), `allow_plaintext_backup` (opt bool, default `false`), `derived` (opt bool, default `false`), `auto_rotate_period` (opt string, e.g. `"720h"`) |
| **Vault** | `POST transit/keys/:name` |
| **Returns** | Confirmation incl. key name, type, and safe-default summary. |
| **Validation** | `validateKeyName(name)`; reject unknown `type` (allowlist). |
| **Safe defaults** | `exportable=false`, `allow_plaintext_backup=false`. |

### 2. `read_transit_key` — WS-A · `read_key.go`

| | |
|---|---|
| **Description** | Read configuration/metadata for a Transit key (type, versions, capabilities). |
| **Params** | `mount` (opt), `name` (req) |
| **Vault** | `GET transit/keys/:name` |
| **Returns** | `type`, list of versions + creation times, `latest_version`, `min_decryption_version`, `supports_encryption/decryption/derivation/signing`, `exportable`. |
| **Annotation** | ReadOnly. |

### 3. `rotate_transit_key` — WS-A · `rotate_key.go`

| | |
|---|---|
| **Description** | Rotate a Transit key to a new version. Old ciphertext stays decryptable. |
| **Params** | `mount` (opt), `name` (req) |
| **Vault** | `POST transit/keys/:name/rotate` |
| **Returns** | New latest version number; reminder that `rewrap_data` upgrades old ciphertext. |

### 4. `encrypt_data` — WS-B · `encrypt_data.go`

| | |
|---|---|
| **Description** | Encrypt data using a Transit key. |
| **Params** | `mount` (opt), `name` (req), `plaintext` (req), `plaintext_is_base64` (opt bool, default `false`), `context` (opt, b64, for derived keys), `key_version` (opt int), `nonce` (opt, b64) |
| **Vault** | `POST transit/encrypt/:name` with `plaintext` base64-encoded |
| **Returns** | `ciphertext` = `vault:vN:...` |
| **Design** | If `plaintext_is_base64=false`, base64-encode the raw input before sending. If `true`, `validateBase64` first. |

### 5. `decrypt_data` — WS-B · `decrypt_data.go`

| | |
|---|---|
| **Description** | Decrypt Vault ciphertext using a Transit key. |
| **Params** | `mount` (opt), `name` (req), `ciphertext` (req), `context` (opt, b64) |
| **Vault** | `POST transit/decrypt/:name` |
| **Returns** | The base64 `plaintext` from Vault **and** the decoded UTF-8 text when valid. |
| **Validation** | `validateCiphertext(ciphertext)` before calling Vault. |

### 6. `rewrap_data` — WS-B · `rewrap_data.go`

| | |
|---|---|
| **Description** | Re-encrypt existing ciphertext with the key's latest version (no plaintext exposed). |
| **Params** | `mount` (opt), `name` (req), `ciphertext` (req), `context` (opt, b64) |
| **Vault** | `POST transit/rewrap/:name` |
| **Returns** | New `ciphertext` (higher `vault:vN:` version). |
| **Validation** | `validateCiphertext`. |

### 7. `generate_hmac` — WS-C · `generate_hmac.go`

| | |
|---|---|
| **Description** | Generate an HMAC for data-integrity verification. |
| **Params** | `mount` (opt), `name` (req), `input` (req), `input_is_base64` (opt bool, default `false`), `algorithm` (opt, default `sha2-256`), `key_version` (opt int) |
| **Vault** | `POST transit/hmac/:name/:algorithm` with base64 `input` |
| **Returns** | `hmac` = `vault:vN:...` |

### 8. `verify_hmac` — WS-C · `verify_hmac.go`

| | |
|---|---|
| **Description** | Verify an HMAC against the original input. |
| **Params** | `mount` (opt), `name` (req), `input` (req), `input_is_base64` (opt bool), `hmac` (req), `algorithm` (opt, default `sha2-256`) |
| **Vault** | `POST transit/verify/:name/:algorithm` |
| **Returns** | Boolean `valid` with a clear human-readable verdict. |

---

## Stretch tools (select)

### 9. `list_transit_keys` — WS-A · `list_keys.go`
`LIST transit/keys` → `keys[]`. Params: `mount` (opt). ReadOnly.

### 10. `sign_data` — WS-C · `sign_data.go`
`POST transit/sign/:name/:hash_algorithm`. Params: `mount`, `name` (req, asymmetric key),
`input` (req, b64), `hash_algorithm` (opt, default `sha2-256`), `signature_algorithm` (opt,
default `pss` for RSA), `key_version` (opt). Returns `signature` = `vault:vN:...`.

### 11. `verify_signature` — WS-C · `verify_signature.go`
`POST transit/verify/:name/:hash_algorithm`. Params: `mount`, `name` (req), `input` (req, b64),
`signature` (req), `hash_algorithm` (opt). Returns `valid` (bool).

### 12. `hash_data` — WS-C · `hash_data.go`
`POST transit/hash/:algorithm`. Params: `algorithm` (opt, default `sha2-256`), `input` (req, b64),
`format` (opt, `hex`|`base64`, default `hex`). Returns `sum`. (Mount-independent — uses default `transit`.)

### 13. `generate_random_bytes` — WS-C · `generate_random_bytes.go`
`POST transit/random/:bytes`. Params: `bytes` (opt int, default 32), `format` (opt, `base64`|`hex`,
default `base64`). Returns `random_bytes`. ReadOnly.

---

## Reference implementation — `create_transit_key.go`

Use this as the canonical template for the other tools.

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

// CreateTransitKey creates a tool for creating a Vault Transit encryption key.
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
                mcp.Description("Key type. Defaults to 'aes256-gcm96'. Use an asymmetric type "+
                    "(e.g. 'ed25519', 'rsa-2048') only if you need signing.")),
            mcp.WithBoolean("exportable",
                mcp.Description("Allow key export. Defaults to false (recommended).")),
            mcp.WithBoolean("allow_plaintext_backup",
                mcp.Description("Allow plaintext backup of the key. Defaults to false.")),
            mcp.WithBoolean("derived",
                mcp.Description("Require a context for encrypt/decrypt (key derivation). Default false.")),
            mcp.WithString("auto_rotate_period",
                mcp.Description("Optional auto-rotation period, e.g. '720h'. Empty disables it.")),
        ),
        Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            return createTransitKeyHandler(ctx, req, logger)
        },
    }
}

func createTransitKeyHandler(ctx context.Context, req mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
    logger.Debug("Handling create_transit_key request")

    args, ok := req.Params.Arguments.(map[string]interface{})
    if !ok {
        return mcp.NewToolResultError("Missing or invalid arguments format"), nil
    }

    mount := resolveMount(args)

    name, err := extractString(args, "name", true)
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }
    if err := validateKeyName(name); err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }

    keyType, _ := extractString(args, "type", false)
    if keyType == "" {
        keyType = "aes256-gcm96"
    }
    // (optional) validate keyType against an allowlist here.

    payload := map[string]interface{}{
        "type":                   keyType,
        "exportable":             extractBool(args, "exportable", false),
        "allow_plaintext_backup": extractBool(args, "allow_plaintext_backup", false),
        "derived":                extractBool(args, "derived", false),
    }
    if p, _ := extractString(args, "auto_rotate_period", false); p != "" {
        payload["auto_rotate_period"] = p
    }

    vault, err := client.GetVaultClientFromContext(ctx, logger)
    if err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("Failed to get Vault client: %v", err)), nil
    }

    if _, err := vault.Logical().Write(transitPath(mount, "keys", name), payload); err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("Failed to create transit key '%s': %v", name, err)), nil
    }

    logger.WithFields(log.Fields{"mount": mount, "name": name, "type": keyType}).
        Info("Created transit key")

    return mcp.NewToolResultText(fmt.Sprintf(
        "Created Transit key '%s' (type=%s, exportable=false) in mount '%s'. "+
            "You can now encrypt data with encrypt_data.", name, keyType, mount)), nil
}
```

## Reference implementation — `encrypt_data.go` (handler core)

```go
func encryptDataHandler(ctx context.Context, req mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
    args, ok := req.Params.Arguments.(map[string]interface{})
    if !ok {
        return mcp.NewToolResultError("Missing or invalid arguments format"), nil
    }
    mount := resolveMount(args)

    name, err := extractString(args, "name", true)
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }
    plaintext, err := extractString(args, "plaintext", true)
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }

    // Agent-friendly: accept raw text and base64-encode it, unless told it's already base64.
    b64 := plaintext
    if extractBool(args, "plaintext_is_base64", false) {
        if err := validateBase64(plaintext); err != nil {
            return mcp.NewToolResultError(err.Error()), nil
        }
    } else {
        b64 = base64.StdEncoding.EncodeToString([]byte(plaintext))
    }

    payload := map[string]interface{}{"plaintext": b64}
    if c, _ := extractString(args, "context", false); c != "" {
        payload["context"] = c
    }
    if v := extractInt(args, "key_version", 0); v > 0 {
        payload["key_version"] = v
    }
    if n, _ := extractString(args, "nonce", false); n != "" {
        payload["nonce"] = n
    }

    vault, err := client.GetVaultClientFromContext(ctx, logger)
    if err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("Failed to get Vault client: %v", err)), nil
    }

    secret, err := vault.Logical().Write(transitPath(mount, "encrypt", name), payload)
    if err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("Failed to encrypt with key '%s': %v", name, err)), nil
    }
    ciphertext, err := dataString(secret, "ciphertext")
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }
    return mcp.NewToolResultText(fmt.Sprintf("Ciphertext: %s", ciphertext)), nil
}
```

## Reference unit test — `create_key_test.go`

```go
// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package transit

import (
    "net/http"
    "testing"

    "github.com/mark3labs/mcp-go/mcp"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestCreateTransitKeyHandler_Success(t *testing.T) {
    logger := newLogger()
    var captured map[string]interface{}

    mux := http.NewServeMux()
    mux.HandleFunc("/v1/transit/keys/customer-data", func(w http.ResponseWriter, r *http.Request) {
        require.Equal(t, http.MethodPut, r.Method) // vault api uses PUT for writes
        decodeBody(r, &captured)
        w.WriteHeader(http.StatusNoContent)
    })

    ctx, cleanup := newTestContext(t, mux)
    defer cleanup()

    req := mcp.CallToolRequest{Params: mcp.CallToolParams{
        Name: "create_transit_key",
        Arguments: map[string]interface{}{"name": "customer-data"},
    }}

    result, err := createTransitKeyHandler(ctx, req, logger)
    require.NoError(t, err)
    require.NotNil(t, result)
    assert.False(t, result.IsError, "expected success: %s", getResultText(result))
    assert.Equal(t, "aes256-gcm96", captured["type"])
    assert.Equal(t, false, captured["exportable"])
}

func TestCreateTransitKeyHandler_MissingName(t *testing.T) {
    logger := newLogger()
    ctx, cleanup := newTestContext(t, http.NewServeMux())
    defer cleanup()

    req := mcp.CallToolRequest{Params: mcp.CallToolParams{
        Name: "create_transit_key", Arguments: map[string]interface{}{},
    }}
    result, err := createTransitKeyHandler(ctx, req, logger)
    require.NoError(t, err)
    assert.True(t, result.IsError)
}
```

> `decodeBody` is a tiny shared helper (add to `transit_test.go`) wrapping
> `json.NewDecoder(r.Body).Decode(&v)`.

## Registration block (orchestrator appends to `tools.go`)

```go
// Tools for Transit encryption-as-a-service
createKey := transit.CreateTransitKey(logger)
hcServer.AddTool(createKey.Tool, createKey.Handler)

readKey := transit.ReadTransitKey(logger)
hcServer.AddTool(readKey.Tool, readKey.Handler)

rotateKey := transit.RotateTransitKey(logger)
hcServer.AddTool(rotateKey.Tool, rotateKey.Handler)

encryptData := transit.EncryptData(logger)
hcServer.AddTool(encryptData.Tool, encryptData.Handler)

decryptData := transit.DecryptData(logger)
hcServer.AddTool(decryptData.Tool, decryptData.Handler)

rewrapData := transit.RewrapData(logger)
hcServer.AddTool(rewrapData.Tool, rewrapData.Handler)

generateHMAC := transit.GenerateHMAC(logger)
hcServer.AddTool(generateHMAC.Tool, generateHMAC.Handler)

verifyHMAC := transit.VerifyHMAC(logger)
hcServer.AddTool(verifyHMAC.Tool, verifyHMAC.Handler)

// Stretch
listKeys := transit.ListTransitKeys(logger)
hcServer.AddTool(listKeys.Tool, listKeys.Handler)
signData := transit.SignData(logger)
hcServer.AddTool(signData.Tool, signData.Handler)
verifySignature := transit.VerifySignature(logger)
hcServer.AddTool(verifySignature.Tool, verifySignature.Handler)
hashData := transit.HashData(logger)
hcServer.AddTool(hashData.Tool, hashData.Handler)
randomBytes := transit.GenerateRandomBytes(logger)
hcServer.AddTool(randomBytes.Tool, randomBytes.Handler)
```

## Constructor name registry (the naming contract)

| Tool name | Constructor | File | Owner |
|-----------|-------------|------|-------|
| `create_transit_key` | `CreateTransitKey` | `create_key.go` | WS-A |
| `read_transit_key` | `ReadTransitKey` | `read_key.go` | WS-A |
| `rotate_transit_key` | `RotateTransitKey` | `rotate_key.go` | WS-A |
| `list_transit_keys` | `ListTransitKeys` | `list_keys.go` | WS-A |
| `encrypt_data` | `EncryptData` | `encrypt_data.go` | WS-B |
| `decrypt_data` | `DecryptData` | `decrypt_data.go` | WS-B |
| `rewrap_data` | `RewrapData` | `rewrap_data.go` | WS-B |
| `generate_hmac` | `GenerateHMAC` | `generate_hmac.go` | WS-C |
| `verify_hmac` | `VerifyHMAC` | `verify_hmac.go` | WS-C |
| `sign_data` | `SignData` | `sign_data.go` | WS-C |
| `verify_signature` | `VerifySignature` | `verify_signature.go` | WS-C |
| `hash_data` | `HashData` | `hash_data.go` | WS-C |
| `generate_random_bytes` | `GenerateRandomBytes` | `generate_random_bytes.go` | WS-C |
