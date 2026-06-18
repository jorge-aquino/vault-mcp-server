# Testing Strategy — Vault Transit Tools

> Adapted from `plan/06-testing-strategy.md`. Describes the test architecture, coverage
> expectations, how to run tests, and the e2e lifecycle outline for the Transit package.

---

## Test pyramid

```
                    ┌─────────────────┐
                    │   e2e tests     │  ← real vault server -dev, Transit lifecycle
                    │ (build tag: e2e)│
                    └────────┬────────┘
               ┌─────────────┴─────────────┐
               │     validation tests       │  ← error paths, input guards (mocked)
               └─────────────┬─────────────┘
        ┌────────────────────┴────────────────────┐
        │             unit tests                   │  ← each handler in isolation (mocked)
        └──────────────────────────────────────────┘
```

| Layer | Scope | Location | Vault |
|-------|-------|----------|-------|
| **Unit** | Each handler in isolation | `pkg/tools/transit/*_test.go` | `httptest` mock |
| **Validation** | Input guards and error paths | Same `*_test.go` files | Mock / none |
| **Integration (e2e)** | Full lifecycle workflow | `e2e/transit_e2e_test.go` | Real `vault server -dev` |
| **Regression** | Existing tools unaffected | `kv/`, `pki/`, `sys/` suites | Mocked |

---

## How to run tests

### Unit tests only (no Vault required)
```bash
go test ./pkg/tools/transit/...
# or via Makefile:
make test-transit
```

### All unit tests including regression
```bash
make test
```

### Transit e2e tests (requires vault server -dev with transit enabled)
```bash
# Prerequisites:
vault server -dev &
export VAULT_ADDR=http://127.0.0.1:8200
export VAULT_TOKEN=<root-token-from-vault-dev>
vault secrets enable transit

# Run e2e:
make test-transit-e2e
# which runs:
# go test -tags=e2e ./e2e/ -run Transit -v
```

### Run with race detector
```bash
go test -race ./pkg/tools/transit/...
```

### Static checks
```bash
gofmt -l ./pkg/tools/transit/   # must print nothing
go vet ./...                     # must be clean
```

---

## Unit test structure

Every unit test file uses the shared helpers from
[`pkg/tools/transit/transit_test.go`](../pkg/tools/transit/transit_test.go):

```go
// newLogger()           — logrus at Error level (suppresses test noise)
// newTestContext(t, mux) — httptest server + Vault client + MCP context
// jsonResponse(w, body) — writes JSON response with Content-Type
// getResultText(result) — extracts text from *mcp.CallToolResult
// decodeBody(r, &v)     — JSON-decodes HTTP request body (for payload assertions)
```

### Mock endpoint patterns

The Vault Go SDK sends writes as HTTP `PUT` (not `POST`) to `/v1/...`:

```go
// Create/update
mux.HandleFunc("/v1/transit/keys/customer-data", func(w http.ResponseWriter, r *http.Request) {
    require.Equal(t, http.MethodPut, r.Method)
    w.WriteHeader(http.StatusNoContent)
})

// Read
mux.HandleFunc("/v1/transit/keys/customer-data", func(w http.ResponseWriter, r *http.Request) {
    require.Equal(t, http.MethodGet, r.Method)
    jsonResponse(w, map[string]interface{}{
        "data": map[string]interface{}{
            "type": "aes256-gcm96",
            "latest_version": 1,
            "min_decryption_version": 1,
            "exportable": false,
        },
    })
})

// Encrypt
mux.HandleFunc("/v1/transit/encrypt/customer-data", func(w http.ResponseWriter, r *http.Request) {
    jsonResponse(w, map[string]interface{}{
        "data": map[string]interface{}{"ciphertext": "vault:v1:abc123=="},
    })
})

// HMAC
mux.HandleFunc("/v1/transit/hmac/customer-data/sha2-256", func(w http.ResponseWriter, r *http.Request) {
    jsonResponse(w, map[string]interface{}{
        "data": map[string]interface{}{"hmac": "vault:v1:hmacvalue=="},
    })
})

// Vault error
mux.HandleFunc("/v1/transit/...", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusInternalServerError)
    jsonResponse(w, map[string]interface{}{"errors": []string{"internal error"}})
})
```

---

## Unit test coverage matrix

The minimum required test cases per tool:

| Tool | Happy path | Missing param | Bad base64 | Bad ciphertext | Vault error |
|------|:----------:|:-------------:|:----------:|:--------------:|:-----------:|
| `create_transit_key` | ✓ | ✓ | — | — | ✓ |
| `read_transit_key` | ✓ | ✓ | — | — | ✓ |
| `rotate_transit_key` | ✓ | ✓ | — | — | ✓ |
| `list_transit_keys` | ✓ | — | — | — | ✓ |
| `encrypt_data` | ✓ | ✓ | ✓ | — | ✓ |
| `decrypt_data` | ✓ | ✓ | — | ✓ | ✓ |
| `rewrap_data` | ✓ | ✓ | — | ✓ | ✓ |
| `generate_hmac` | ✓ | ✓ | ✓ | — | ✓ |
| `verify_hmac` | ✓ (valid+invalid) | ✓ | ✓ | — | ✓ |
| `sign_data` | ✓ | ✓ | ✓ | — | ✓ |
| `verify_signature` | ✓ | ✓ | ✓ | — | ✓ |
| `hash_data` | ✓ | ✓ | ✓ | — | ✓ |
| `generate_random_bytes` | ✓ | — | — | — | ✓ |

### Important: assert Vault request payloads in happy-path tests

Happy-path tests must assert what the tool *sends* to Vault, not just what it returns:

```go
var captured map[string]interface{}
mux.HandleFunc("/v1/transit/keys/customer-data", func(w http.ResponseWriter, r *http.Request) {
    decodeBody(r, &captured)
    w.WriteHeader(http.StatusNoContent)
})
// ... call handler ...
assert.Equal(t, "aes256-gcm96", captured["type"])
assert.Equal(t, false, captured["exportable"])
```

This catches bugs where safe defaults are not applied (e.g. exportable not set to false).

### Pre-call validation test pattern

For ciphertext validation, verify the Vault endpoint is **never called** on bad input:

```go
handlerCalled := false
mux.HandleFunc("/v1/transit/decrypt/customer-data", func(w http.ResponseWriter, r *http.Request) {
    handlerCalled = true
    w.WriteHeader(http.StatusNoContent)
})
// Call with bad ciphertext
result, err := decryptDataHandler(ctx, req, logger)
require.NoError(t, err)
assert.True(t, result.IsError)
assert.False(t, handlerCalled, "Vault should not be called for invalid ciphertext")
```

---

## e2e lifecycle test outline

File: `e2e/transit_e2e_test.go`  
Build tag: `//go:build e2e`

```
 1. create_transit_key("e2e-key")                       → success
 2. read_transit_key("e2e-key")                         → latest_version == 1
 3. encrypt_data("e2e-key", plaintext="hello world")    → ct1 starts with "vault:v1:"
 4. decrypt_data("e2e-key", ct1)                        → plaintext == "hello world"  (round-trip)
 5. rotate_transit_key("e2e-key")                       → latest_version == 2
 6. encrypt_data("e2e-key", plaintext="hello world")    → ct2 starts with "vault:v2:"
 7. rewrap_data("e2e-key", ct1)                         → ct3 starts with "vault:v2:"
 8. decrypt_data("e2e-key", ct1)                        → still "hello world" (key history preserved)
 9. decrypt_data("e2e-key", ct3)                        → still "hello world" (rewrapped ciphertext)
10. generate_hmac("e2e-key", "hello world")             → hmac starts with "vault:v"
11. verify_hmac("e2e-key", "hello world", hmac)         → valid == true
12. verify_hmac("e2e-key", "Hello world", hmac)         → valid == false  (tampered input)
```

Key assertions for the demo narrative:
- **Round-trip equality**: `decrypt(encrypt(x)) == x`
- **Version bump after rotate**: `latest_version == 2` after rotation
- **Rewrapped ciphertext version**: starts with `vault:v2:` not `vault:v1:`
- **Key history preserved**: v1 ciphertext still decryptable after rotation
- **HMAC sensitivity**: single character change invalidates HMAC

---

## Validation test assertions

Confirm human-readable error messages for all input guard cases:

| Input error | Expected message pattern |
|------------|--------------------------|
| Empty key name | `parameter 'name' must not be empty` |
| Key name with `/` | `key name must not contain '/'` |
| Invalid base64 | `value is not valid base64` |
| Missing `vault:v` prefix | `invalid ciphertext: expected a 'vault:v<version>:...' value` |
| Missing required param | `missing required parameter '<name>'` |
| Vault 404 | `Failed to ... key '...': ...` |
| Vault 403 | surfaced as tool error with status |

---

## Regression

After adding Transit tools, verify all existing suites still pass:

```bash
make test   # runs kv, pki, sys, and transit unit tests
```

The Transit import must not introduce build cycles. The `transit` package imports:
- `github.com/hashicorp/vault-mcp-server/pkg/client`
- `github.com/hashicorp/vault-mcp-server/pkg/utils`
- `github.com/hashicorp/vault/api`
- `github.com/mark3labs/mcp-go/mcp`
- `github.com/mark3labs/mcp-go/server`
- `github.com/sirupsen/logrus`

None of these create a cycle with the existing tool packages.

---

## CI integration

- **Unit + validation tests**: run on every push (no Vault required — all mocked)
- **e2e tests**: run in a job that starts `vault server -dev`, enables transit, sets
  `VAULT_ADDR`/`VAULT_TOKEN`, then runs `make test-transit-e2e`
- **Static checks**: `gofmt -l` and `go vet ./...` run as a lint gate
- **Copyright header check**: grep for `SPDX-License-Identifier` in all new `.go` files

See `.github/workflows/unit_test.yml` and `.github/workflows/e2e_test.yml` for CI config.
