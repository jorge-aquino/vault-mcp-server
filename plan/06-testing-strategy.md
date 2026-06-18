# 06 — Testing Strategy (WS-E)

> Owned by **WS-E** with unit tests authored alongside each tool by WS-A/B/C. The goal: prove
> the Transit tools work independently *and* as a complete workflow, without breaking existing
> server behavior.

## Test pyramid

| Layer | Scope | Where | Vault |
|-------|-------|-------|-------|
| **Unit** | each handler in isolation | `pkg/tools/transit/*_test.go` (with each tool) | mocked (`httptest`) |
| **Validation** | input guards & error paths | same `*_test.go` | mocked / none |
| **Integration (e2e)** | full lifecycle workflow | `e2e/transit_e2e_test.go` | real `vault server -dev` |
| **Regression** | existing tools unaffected | existing `kv/pki/sys` suites | mocked |

## 1. Unit tests (authored by tool owners)

For every tool, table-driven tests using the shared helpers (`newLogger`, `newTestContext`,
`jsonResponse`, `getResultText`, `decodeBody`):

- **Happy path** — assert the **request payload** sent to Vault is correct (e.g. `type` defaults
  to `aes256-gcm96`, `exportable=false`; `plaintext` is base64-encoded), and the success text
  contains the expected value (ciphertext/version/valid).
- **Missing required param** — returns `IsError` result.
- **Bad base64** (encrypt/hmac/sign with `*_is_base64=true`) — rejected before any Vault call.
- **Malformed ciphertext** (decrypt/rewrap without `vault:v` prefix) — rejected pre-call.
- **Vault error** — mock returns 4xx/5xx; handler returns a clear `NewToolResultError`.

Mock endpoints follow the API map in [01-system-architecture.md](01-system-architecture.md):

```go
mux.HandleFunc("/v1/transit/keys/customer-data", ...)   // create/read
mux.HandleFunc("/v1/transit/encrypt/customer-data", ...) // encrypt
mux.HandleFunc("/v1/transit/decrypt/customer-data", ...) // decrypt
mux.HandleFunc("/v1/transit/keys/customer-data/rotate", ...)
mux.HandleFunc("/v1/transit/rewrap/customer-data", ...)
mux.HandleFunc("/v1/transit/hmac/customer-data/sha2-256", ...)
mux.HandleFunc("/v1/transit/verify/customer-data/sha2-256", ...)
```

> Note: the Vault Go API issues writes as HTTP **PUT** (not POST) to `/v1/...`. Assert
> accordingly in mocks.

### Minimum unit coverage matrix

| Tool | Happy | Missing param | Bad base64 | Bad ciphertext | Vault error |
|------|:-----:|:-------------:|:----------:|:--------------:|:-----------:|
| create_transit_key | ✓ | ✓ | — | — | ✓ |
| read_transit_key | ✓ | ✓ | — | — | ✓ |
| rotate_transit_key | ✓ | ✓ | — | — | ✓ |
| encrypt_data | ✓ | ✓ | ✓ | — | ✓ |
| decrypt_data | ✓ | ✓ | — | ✓ | ✓ |
| rewrap_data | ✓ | ✓ | — | ✓ | ✓ |
| generate_hmac | ✓ | ✓ | ✓ | — | ✓ |
| verify_hmac | ✓ (valid+invalid) | ✓ | ✓ | — | ✓ |
| stretch tools | ✓ | ✓ | ✓ where applicable | — | ✓ |

## 2. Integration / e2e (`e2e/transit_e2e_test.go`)

Runs the **real** lifecycle against `vault server -dev` with Transit enabled. Gate behind an env
flag so unit CI stays hermetic.

```go
//go:build e2e
// +build e2e

package e2e

// Requires: VAULT_ADDR + VAULT_TOKEN set, `vault secrets enable transit` done.
// Flow:
//   1. create_transit_key("e2e-key")
//   2. read_transit_key -> latest_version == 1
//   3. encrypt_data("hello world") -> ct1 starts "vault:v1:"
//   4. decrypt_data(ct1) -> "hello world"   (round-trip)
//   5. rotate_transit_key -> latest_version == 2
//   6. rewrap_data(ct1) -> ct2 starts "vault:v2:"
//   7. decrypt_data(ct2) -> "hello world"   (still decryptable)
//   8. generate_hmac("hello world") -> hmac starts "vault:v"
//   9. verify_hmac(input, hmac) -> valid == true
//  10. verify_hmac(tampered input, hmac) -> valid == false
```

Assertions that matter for the demo narrative:
- Round-trip equality (decrypt(encrypt(x)) == x).
- Version bump after rotate; rewrapped ciphertext is `vault:v2:`.
- Old ciphertext still decrypts after rotation (key history preserved).
- HMAC verify true for original, false for tampered input.

## 3. Validation tests (explicit error UX)

Confirm friendly, actionable messages for:
- invalid base64 plaintext/input,
- missing/empty key name, empty mount handled by default,
- malformed ciphertext (no `vault:v` prefix),
- unsupported algorithm string,
- unknown key (Vault 404 surfaced clearly).

## 4. Regression

- `make test` must keep all existing `kv`, `pki`, `sys` tests green.
- New `transit` import must not introduce build cycles or break `InitTools`.

## 5. Static checks

```bash
gofmt -l ./pkg/tools/transit            # must print nothing
go vet ./...                            # clean
# copyright/SPDX header present on every new .go file (grep check in CI)
```

## 6. Makefile additions (WS-E)

```make
.PHONY: test-transit test-transit-e2e

test-transit:
	go test ./pkg/tools/transit/...

test-transit-e2e:
	go test -tags=e2e ./e2e/ -run Transit -v
```

Wire `test-transit` into the existing `make test`, and `test-transit-e2e` into `make test-e2e`.

## 7. CI considerations

- Unit + validation + regression run on every push (no Vault needed — all mocked).
- e2e runs in a job that boots `vault server -dev` (or the dev container) and sets
  `VAULT_ADDR`/`VAULT_TOKEN`, then `make test-transit-e2e`.
- Add the header/format/vet checks as a lint gate.

## Definition of done (testing)

- [ ] Every tool has unit tests meeting the coverage matrix.
- [ ] e2e lifecycle passes against real Vault dev.
- [ ] Validation/error messages verified.
- [ ] Existing suites unaffected (`make test` green).
- [ ] `gofmt`/`go vet`/header checks pass.
