# Transit Tool Examples

One runnable example for each of the 13 Vault Transit MCP tools. All examples use the key name
`customer-data` and realistic inputs. Use these as a reference for Bob prompts or direct tool calls.

> **Note:** The "Expected output" blocks below are illustrative examples. Exact output text may
> vary slightly between Vault versions and server configurations.

---

## Key Management Tools

### `create_transit_key`

**Description:** Create a named encryption key in the Vault Transit engine.

**Example Bob prompt:**
> Create a Vault Transit encryption key named `customer-data` of type aes256-gcm96.

**Example arguments:**
```json
{
  "name": "customer-data",
  "type": "aes256-gcm96",
  "exportable": false,
  "allow_plaintext_backup": false
}
```

**Expected output:**
```
Created Transit key 'customer-data' (type=aes256-gcm96, exportable=false) in mount 'transit'.
You can now encrypt data with encrypt_data.
```

---

### `read_transit_key`

**Description:** Read the configuration and metadata for a Transit key, including its type,
key versions, and supported capabilities.

**Example Bob prompt:**
> Read the metadata for the `customer-data` Transit key to see its current version and capabilities.

**Example arguments:**
```json
{
  "name": "customer-data"
}
```

**Expected output:**
```
Transit key 'customer-data':
  Type: aes256-gcm96
  Latest version: 1
  Min decryption version: 1
  Exportable: false
  Supports encryption: true
  Supports decryption: true
  Supports signing: false
  Supports derivation: true
  Key versions: {1: created 2025-01-15T10:00:00Z}
```

---

### `rotate_transit_key`

**Description:** Rotate the Transit key to create a new version. Existing ciphertext remains
decryptable. New encryptions use the latest version.

**Example Bob prompt:**
> Rotate the `customer-data` key to create a new version.

**Example arguments:**
```json
{
  "name": "customer-data"
}
```

**Expected output:**
```
Rotated Transit key 'customer-data'. New latest version: 2.
Existing ciphertext encrypted with version 1 is still decryptable.
Use rewrap_data to upgrade old ciphertext to the latest version.
```

---

### `list_transit_keys`

**Description:** List all key names in the Transit secrets engine.

**Example Bob prompt:**
> List all Transit keys in the default transit mount.

**Example arguments:**
```json
{}
```

**Expected output:**
```
Transit keys in mount 'transit':
  - customer-data
  - api-signing-key
  - backup-encryption
```

---

## Encryption Tools

### `encrypt_data`

**Description:** Encrypt data using a Transit key. Returns a Vault ciphertext string that
encodes the key version used for encryption.

**Example Bob prompt:**
> Encrypt the string "Alice's account balance is $12,450.00" using the `customer-data` key.

**Example arguments:**
```json
{
  "name": "customer-data",
  "plaintext": "Alice's account balance is $12,450.00"
}
```

**Expected output:**
```
Ciphertext: vault:v1:8SDd3WHDOjf7mq69CyCqYjBk5S/OtGWfXKMuMoFEsRa4...
```

---

### `decrypt_data`

**Description:** Decrypt Vault ciphertext using a Transit key. Returns both the base64 plaintext
and the decoded UTF-8 text.

**Example Bob prompt:**
> Decrypt this ciphertext using the `customer-data` key: `vault:v1:8SDd3WHDOjf7mq69CyCqYjBk5S/OtGWfXKMuMoFEsRa4...`

**Example arguments:**
```json
{
  "name": "customer-data",
  "ciphertext": "vault:v1:8SDd3WHDOjf7mq69CyCqYjBk5S/OtGWfXKMuMoFEsRa4..."
}
```

**Expected output:**
```
Plaintext (base64): QWxpY2UncyBhY2NvdW50IGJhbGFuY2UgaXMgJDEyLDQ1MC4wMA==
Decoded: Alice's account balance is $12,450.00
```

---

### `rewrap_data`

**Description:** Re-encrypt ciphertext with the key's latest version without exposing the
plaintext. Use this after key rotation to upgrade stored ciphertext.

**Example Bob prompt:**
> Rewrap this v1 ciphertext to the current key version: `vault:v1:8SDd3WHDOjf7mq69...`

**Example arguments:**
```json
{
  "name": "customer-data",
  "ciphertext": "vault:v1:8SDd3WHDOjf7mq69CyCqYjBk5S/OtGWfXKMuMoFEsRa4..."
}
```

**Expected output:**
```
Rewrapped ciphertext (now at vault:v2:): vault:v2:dUI9iegXADNTJQ7kGxmBzrP...
The ciphertext has been upgraded from version 1 to version 2 without exposing the plaintext.
```

---

## Integrity and Cryptography Tools

### `generate_hmac`

**Description:** Generate a keyed HMAC for data integrity verification. The HMAC is tied to
the key version at generation time.

**Example Bob prompt:**
> Generate an HMAC for the string "transaction:id=abc123,amount=450.00,ts=1705315200" using
> the `customer-data` key with SHA2-256.

**Example arguments:**
```json
{
  "name": "customer-data",
  "input": "transaction:id=abc123,amount=450.00,ts=1705315200",
  "algorithm": "sha2-256"
}
```

**Expected output:**
```
HMAC: vault:v2:MlFa3M6+kLmqOV1ZqKpjhN8jR0bYVfZW4Xk...
```

---

### `verify_hmac`

**Description:** Verify that an HMAC matches the original input. Returns a clear boolean
verdict with a human-readable explanation.

**Example Bob prompt:**
> Verify this HMAC for the string "transaction:id=abc123,amount=450.00,ts=1705315200":
> `vault:v2:MlFa3M6+kLmqOV1ZqKpjhN8jR0bYVfZW4Xk...`

**Example arguments:**
```json
{
  "name": "customer-data",
  "input": "transaction:id=abc123,amount=450.00,ts=1705315200",
  "hmac": "vault:v2:MlFa3M6+kLmqOV1ZqKpjhN8jR0bYVfZW4Xk...",
  "algorithm": "sha2-256"
}
```

**Expected output (match):**
```
HMAC verified successfully: the input matches the HMAC.
```

**Expected output (mismatch):**
```
HMAC verification failed: the input does not match the provided HMAC.
```

---

### `sign_data`

**Description:** Sign data using an asymmetric Transit key (ed25519, ecdsa, or RSA). Requires
a key created with an asymmetric key type.

**Example Bob prompt:**
> Sign the base64-encoded string "dGhlIHF1aWNrIGJyb3duIGZveA==" using the `api-signing-key`
> key with SHA2-256.

**Example arguments:**
```json
{
  "name": "api-signing-key",
  "input": "dGhlIHF1aWNrIGJyb3duIGZveA==",
  "input_is_base64": true,
  "hash_algorithm": "sha2-256"
}
```

**Expected output:**
```
Signature: vault:v1:MEUCIQDy4jVhB7FaV/i3rG1...
```

> **Note:** `api-signing-key` must be created with `type=ed25519` or another asymmetric type.
> Using `aes256-gcm96` will return a Vault error: "signing not supported for key type".

---

### `verify_signature`

**Description:** Verify a digital signature against the original input using an asymmetric
Transit key.

**Example Bob prompt:**
> Verify this signature for the input "dGhlIHF1aWNrIGJyb3duIGZveA==" using `api-signing-key`:
> `vault:v1:MEUCIQDy4jVhB7FaV/i3rG1...`

**Example arguments:**
```json
{
  "name": "api-signing-key",
  "input": "dGhlIHF1aWNrIGJyb3duIGZveA==",
  "input_is_base64": true,
  "signature": "vault:v1:MEUCIQDy4jVhB7FaV/i3rG1...",
  "hash_algorithm": "sha2-256"
}
```

**Expected output:**
```
Signature valid: true
The input matches the signature generated by key 'api-signing-key'.
```

---

### `hash_data`

**Description:** Hash data using a Vault-managed algorithm. Does not require a key — useful
for deterministic fingerprinting.

**Example Bob prompt:**
> Hash the string "the quick brown fox" using SHA2-256 and return the hex digest.

**Example arguments:**
```json
{
  "input": "the quick brown fox",
  "algorithm": "sha2-256",
  "format": "hex"
}
```

**Expected output:**
```
Hash (sha2-256, hex): 9ecb36561341d18eb65484e833efea61edc74b84cf5e6ae1b81c63533e25fc8
```

---

### `generate_random_bytes`

**Description:** Generate cryptographically secure random bytes. Useful for nonces, IVs, and
key material generation outside Vault.

**Example Bob prompt:**
> Generate 32 cryptographically secure random bytes in base64 format.

**Example arguments:**
```json
{
  "bytes": 32,
  "format": "base64"
}
```

**Expected output:**
```
Random bytes (32, base64): 7Xk3mP9qR2vYnL8wJdF5tA6oGzHcBsNi...
```

---

## Full reference table

| Tool | Required params | Optional params | Returns |
|------|----------------|-----------------|---------|
| `create_transit_key` | `name` | `mount`, `type`, `exportable`, `allow_plaintext_backup`, `derived`, `auto_rotate_period` | Confirmation with type and defaults |
| `read_transit_key` | `name` | `mount` | Type, versions, capabilities |
| `rotate_transit_key` | `name` | `mount` | New latest version |
| `list_transit_keys` | — | `mount` | List of key names |
| `encrypt_data` | `name`, `plaintext` | `mount`, `plaintext_is_base64`, `context`, `key_version`, `nonce` | `vault:vN:...` ciphertext |
| `decrypt_data` | `name`, `ciphertext` | `mount`, `context` | Base64 plaintext + decoded text |
| `rewrap_data` | `name`, `ciphertext` | `mount`, `context` | New `vault:vN:...` ciphertext |
| `generate_hmac` | `name`, `input` | `mount`, `input_is_base64`, `algorithm`, `key_version` | `vault:vN:...` HMAC |
| `verify_hmac` | `name`, `input`, `hmac` | `mount`, `input_is_base64`, `algorithm` | Boolean verdict + message |
| `sign_data` | `name`, `input` | `mount`, `input_is_base64`, `hash_algorithm`, `signature_algorithm`, `key_version` | `vault:vN:...` signature |
| `verify_signature` | `name`, `input`, `signature` | `mount`, `input_is_base64`, `hash_algorithm` | Boolean verdict + message |
| `hash_data` | `input` | `mount`, `algorithm`, `format` | Hex or base64 digest |
| `generate_random_bytes` | — | `mount`, `bytes`, `format` | Random bytes in requested format |

For full parameter descriptions, see `pkg/tools/transit/` source files or run the MCP
Inspector against the running server.
