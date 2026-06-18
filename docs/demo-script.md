# Demo Script — Vault Transit via Bob

A narrated 8-step live demo running the full Vault Transit lifecycle through Bob in VS Code.
Estimated run time: 4–5 minutes.

---

## Pre-requisites

Before the demo, complete all of these:

```bash
# 1. Start Vault in dev mode
vault server -dev
# Note the root token from the output, e.g.: Root Token: hvs.xxx

# In a new terminal:
export VAULT_ADDR="http://127.0.0.1:8200"
export VAULT_TOKEN="hvs.xxx"  # replace with your root token

# 2. Enable the Transit secrets engine
vault secrets enable transit
# Expected: Success! Enabled the transit secrets engine at: transit/

# 3. Build the vault-mcp-server
make build

# 4. Start the server (stdio mode, used by VS Code)
./vault-mcp-server
# or HTTP mode for MCP Inspector fallback:
# VAULT_ADDR=http://127.0.0.1:8200 VAULT_TOKEN=hvs.xxx ./vault-mcp-server http --transport-port 8080
```

Verify Bob is connected: open VS Code, confirm the `vault-mcp-server` MCP server shows as active
in the Bob/Copilot Chat panel. Optionally switch to the **Vault Transit Security** custom mode
(`.github/agents/vault-transit.agent.md`) for the guarded persona.

**Fallback:** If VS Code has issues, use the MCP Inspector:
```bash
npx @modelcontextprotocol/inspector http://localhost:8080/mcp
```

---

## Step 1 — Create the encryption key

**Bob prompt:**
> Create a Transit encryption key named `customer-data` using the vault-mcp-server.

**Expected Bob behavior:**
- Calls `create_transit_key` with `{"name": "customer-data"}`
- Returns: `"Created Transit key 'customer-data' (type=aes256-gcm96, exportable=false) in mount 'transit'. You can now encrypt data with encrypt_data."`

**Narration notes:**
- Point out `type=aes256-gcm96` — authenticated symmetric encryption, the safe default
- Point out `exportable=false` — Vault holds the key material; it never leaves Vault
- "No key file on disk, no environment variable, no secrets manager except Vault itself"

---

## Step 2 — Read the key metadata

**Bob prompt:**
> Read the metadata for the `customer-data` Transit key.

**Expected Bob behavior:**
- Calls `read_transit_key` with `{"name": "customer-data"}`
- Returns type, latest_version: 1, supports_encryption: true, supports_decryption: true, exportable: false

**Narration notes:**
- "Version 1 — this key has never been rotated"
- "Read-only operation — Bob knows this is safe to call multiple times without side effects"
- Show the `supports_encryption` and `supports_signing` flags — the key's capabilities

---

## Step 3 — Encrypt data

**Bob prompt:**
> Encrypt the text "the quick brown fox" using the `customer-data` Transit key.

**Expected Bob behavior:**
- Calls `encrypt_data` with `{"name": "customer-data", "plaintext": "the quick brown fox"}`
- Auto-encodes the plaintext to base64 before sending to Vault
- Returns ciphertext starting with `vault:v1:`

**Expected output:**
```
Ciphertext: vault:v1:8SDd3WHDOjf7mq69CyCqYjBk5S/OtGWfXKMuMoFEsRa...
```

**Narration notes:**
- Point out the `vault:v1:` prefix — this encodes the key version used for encryption
- "The base64 encoding of the plaintext happens inside the tool — Bob handles this automatically"
- **Copy the ciphertext** — you will use it in Steps 4, 6, and 7

---

## Step 4 — Decrypt the ciphertext

**Bob prompt:**
> Decrypt that ciphertext using the `customer-data` key.

**Expected Bob behavior:**
- Calls `decrypt_data` with the ciphertext from Step 3
- Validates the `vault:v` prefix before calling Vault
- Returns the original plaintext

**Expected output:**
```
Plaintext (base64): dGhlIHF1aWNrIGJyb3duIGZveA==
Decoded: the quick brown fox
```

**Narration notes:**
- "Round-trip verified — encrypt then decrypt recovers the original text"
- Point out the validation: "Bob checked the ciphertext format before making the Vault call"

---

## Step 5 — Rotate the key

**Bob prompt:**
> Rotate the `customer-data` Transit key.

**Expected Bob behavior:**
- Calls `rotate_transit_key` with `{"name": "customer-data"}`
- Reads the key metadata after rotation
- Returns the new latest version (2)

**Expected output:**
```
Rotated Transit key 'customer-data'. New latest version: 2.
Existing ciphertext encrypted with version 1 is still decryptable.
Use rewrap_data to upgrade old ciphertext to the latest version.
```

**Narration notes:**
- "Key rotation is non-destructive — existing ciphertext still works"
- "New encryptions will automatically use version 2"
- "The v1 ciphertext from Step 3 is still valid — we will prove this in the next step"

---

## Step 6 — Rewrap the old ciphertext

**Bob prompt:**
> Rewrap the v1 ciphertext from Step 3 to the latest key version using `rewrap_data`.

**Expected Bob behavior:**
- Calls `rewrap_data` with the v1 ciphertext
- Returns a new ciphertext starting with `vault:v2:`

**Expected output:**
```
Rewrapped ciphertext (now at vault:v2:): vault:v2:dUI9iegXADNTJQ...
The ciphertext has been upgraded from version 1 to version 2 without exposing the plaintext.
```

**Narration notes:**
- "The ciphertext went from `vault:v1:` to `vault:v2:` — upgrade complete"
- "Rewrap never exposes the plaintext — Vault decrypts and re-encrypts entirely internally"
- "This is why you use rewrap, not decrypt + encrypt — one Vault call, no plaintext crossing the API"

---

## Step 7 — Generate an HMAC

**Bob prompt:**
> Generate an HMAC for the text "the quick brown fox" using the `customer-data` key.

**Expected Bob behavior:**
- Calls `generate_hmac` with `{"name": "customer-data", "input": "the quick brown fox"}`
- Returns HMAC starting with `vault:v2:`

**Expected output:**
```
HMAC: vault:v2:MlFa3M6+kLmqOV1ZqKpjhN8jR0bYVfZW...
```

**Narration notes:**
- "The HMAC version reflects the current key version (v2) after rotation"
- "Same key, different operation — this proves data integrity, not just confidentiality"

---

## Step 8 — Verify the HMAC

**Bob prompt:**
> Verify the HMAC from Step 7 against the original input "the quick brown fox".
> Then verify it against the tampered input "the quick brown Fox" (capital F) to show
> what happens with modified data.

**Expected Bob behavior — first call:**
- Calls `verify_hmac` with original input + HMAC
- Returns: `"HMAC verified successfully: the input matches the HMAC."`

**Expected Bob behavior — second call:**
- Calls `verify_hmac` with tampered input + same HMAC  
- Returns: `"HMAC verification failed: the input does not match the provided HMAC."`

**Narration notes:**
- "One character change — capital F — invalidates the HMAC. Tampered data is detected."
- "Bob validated inputs, called Vault, and interpreted the result — all in one step"
- Close: "This is the full Transit lifecycle. Bob isn't just explaining this — it executed it."

---

## Closing talking points

1. **8 tool calls, zero raw Vault API** — Bob handled base64 encoding, ciphertext validation, and error translation
2. **Safe defaults throughout** — key non-exportable, aes256-gcm96, no plaintext key material ever surfaced
3. **The agentic layer** — show `.github/agents/vault-transit.agent.md`, the path-scoped instructions, and the prompt library
4. **Replicable** — show `docs/add-a-new-vault-engine.md` and explain the playbook transfers to any Vault engine
