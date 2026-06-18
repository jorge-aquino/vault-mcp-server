---
mode: agent
description: Generate httptest-mocked unit tests for a Transit tool.
---

# Write unit tests for a Vault Transit tool

You are writing tests for a `pkg/tools/transit/` handler function. The tests must be hermetic —
no real Vault, no network calls outside the httptest server.

## Setup

Use the shared helpers from `transit_test.go`:
- `newLogger()` — returns a logrus logger at Error level (suppresses noise)
- `newTestContext(t, mux)` — starts an httptest server, wires a Vault client to it, returns `(ctx, cleanup)`
- `jsonResponse(w, body)` — writes a JSON body with correct Content-Type
- `getResultText(result)` — extracts the text from a `*mcp.CallToolResult`
- `decodeBody(r, &v)` — decodes the HTTP request body (for asserting Vault payloads)

## Required test cases

Cover all of the following that apply to the tool under test:

### 1. Happy path
- Register the mock Vault endpoint (e.g. `/v1/transit/encrypt/customer-data`)
- Use `decodeBody` to capture the request payload sent to Vault
- Assert `result.IsError == false`
- Assert the response text contains the expected key value (e.g. ciphertext starts with `vault:v`)
- Assert the Vault request payload has correct values (e.g. `type == "aes256-gcm96"`, `exportable == false`)

### 2. Missing required parameter
- Pass an empty `Arguments` map (or omit the required field)
- Assert `result.IsError == true`
- No Vault endpoint should be registered (the handler must fail before calling Vault)

### 3. Invalid base64 (tools with `*_is_base64=true` input)
- Pass `plaintext_is_base64: true` (or `input_is_base64: true`) with a non-base64 string like `"not!!base64"`
- Assert `result.IsError == true`
- Assert error text mentions base64

### 4. Malformed ciphertext (decrypt_data, rewrap_data)
- Pass a `ciphertext` value without the `vault:v` prefix (e.g. `"badciphertext"`)
- Assert `result.IsError == true`
- Assert the handler rejects it before calling Vault (no endpoint hit)

### 5. Vault returns an error
- Register the mock endpoint to return `http.StatusInternalServerError` with a JSON error body
- Assert `result.IsError == true`
- Assert the error text is human-readable (not a raw Go error dump)

## Mock endpoint patterns

```go
// Write/create endpoints (Vault Go SDK uses PUT)
mux.HandleFunc("/v1/transit/keys/customer-data", func(w http.ResponseWriter, r *http.Request) {
    require.Equal(t, http.MethodPut, r.Method)
    w.WriteHeader(http.StatusNoContent)
})

// Read endpoint
mux.HandleFunc("/v1/transit/keys/customer-data", func(w http.ResponseWriter, r *http.Request) {
    require.Equal(t, http.MethodGet, r.Method)
    jsonResponse(w, map[string]interface{}{
        "data": map[string]interface{}{"type": "aes256-gcm96", "latest_version": 1},
    })
})

// Encrypt with response
mux.HandleFunc("/v1/transit/encrypt/customer-data", func(w http.ResponseWriter, r *http.Request) {
    jsonResponse(w, map[string]interface{}{
        "data": map[string]interface{}{"ciphertext": "vault:v1:abc123=="},
    })
})

// Vault error
mux.HandleFunc("/v1/transit/...", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusInternalServerError)
    jsonResponse(w, map[string]interface{}{"errors": []string{"internal error"}})
})
```

## File header

Every test file must start with:
```go
// Copyright IBM Corp. 2025, 2026
// SPDX-License-Identifier: MPL-2.0
```

## Validation

After writing:
```bash
go test ./pkg/tools/transit/... -run TestYourTool -v
```

All cases must pass with no data race warnings (`go test -race ./pkg/tools/transit/...`).
