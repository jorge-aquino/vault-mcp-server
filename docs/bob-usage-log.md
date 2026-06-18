# Bob Usage Log

> A living log of how Bob was used across the SDLC for the Vault Transit capability suite.
> Entries are tagged by phase and include the actual prompt used. This is graded evidence — be specific.

---

## Planning

**Date:** 2025-01-15
**Prompt:**
> "Help us choose the best Vault secrets engine to implement for a Bob-a-thon project. Compare
> Transit, KV v2 advanced operations, PKI, and dynamic database secrets based on: usefulness to
> security teams, technical complexity, demo quality in 5 minutes, and how well it showcases
> agentic workflows."

**Outcome:** Selected Transit. Bob's analysis: Transit has the clearest end-to-end lifecycle
(create → encrypt → decrypt → rotate → rewrap → verify), is self-contained (only needs vault
server -dev), and the encryption/decryption round-trip is immediately visible in a demo.
Dynamic secrets require more infrastructure setup; PKI is already partially implemented.
Decision recorded in `plan/00-overview-and-win-strategy.md`.

---

**Date:** 2025-01-15
**Prompt:**
> "Design a safe, demo-friendly workflow for Vault Transit that shows the full key lifecycle
> including rotation and ciphertext migration, without exposing raw key material."

**Outcome:** The 8-step demo flow: create → read → encrypt → decrypt → rotate → rewrap → HMAC
→ verify. Bob identified that rewrap (not decrypt+encrypt) is the correct operation for
ciphertext migration. This became the canonical workflow in the demo and docs.

---

## Design

**Date:** 2025-01-16
**Prompt:**
> "Break down Vault Transit support into MCP tools with clear names, parameters, return values,
> and Vault API paths. Follow the existing tool pattern in pkg/tools/kv/write_secret.go. Include
> which inputs need base64 encoding, which need the vault:v prefix validation, and what safe
> defaults to apply."

**Outcome:** 8 core tools + 5 stretch tools defined with exact constructor names, file names,
params, and Vault paths. Became `plan/04-tool-specifications.md`. Bob identified the ciphertext
validation requirement and the auto-encode-unless-flagged pattern for plaintext inputs.

---

**Date:** 2025-01-16
**Prompt:**
> "Design the shared helpers file for the transit package. What functions do all 13 tools need
> in common? What validation logic should be centralized to avoid duplication?"

**Outcome:** `transit_helpers.go` with `resolveMount`, `transitPath`, `extractString`,
`extractBool`, `extractInt`, `validateKeyName`, `validateBase64`, `validateCiphertext`, and
`dataString`. Bob noted that `extractInt` needs to handle `float64` from JSON unmarshalling,
which was an important edge case.

---

**Date:** 2025-01-16
**Prompt:**
> "Design the test helper infrastructure for the transit package. What shared helpers do all
> the unit tests need? Model after the existing kv_test.go helpers but make them transit-specific."

**Outcome:** `transit_test.go` with `fakeSession`, `newLogger`, `newTestContext`, `jsonResponse`,
`getResultText`, and `decodeBody`. Bob's key insight: `decodeBody` is needed to assert Vault
request payloads (not just responses), which is the most important unit test assertion.

---

## Implementation

**Date:** 2025-01-17
**Prompt:** (using `.github/prompts/add-transit-tool.prompt.md`)
> "Add a new Transit tool named `rotate_transit_key` following plan/01 + plan/04. It should
> POST to transit/keys/:name/rotate, require only the key name, and return the new latest
> version. Read the key metadata after rotation to surface the version number."

**Outcome:** `rotate_key.go` + `rotate_key_test.go` with happy path and missing-name test.
Bob correctly identified the rotate endpoint as a POST with an empty payload and used the
post-rotate read pattern to surface the new version number.

---

**Date:** 2025-01-17
**Prompt:** (using `.github/prompts/add-transit-tool.prompt.md`)
> "Add the `encrypt_data` tool. Accept raw plaintext from the user and base64-encode it
> automatically. If plaintext_is_base64 is true, validate the base64 first. Return the vault:v
> ciphertext. Follow plan/04 for the full parameter list including context, key_version, nonce."

**Outcome:** `encrypt_data.go` with the auto-encode pattern. Bob correctly wired up the
optional `context`, `key_version`, and `nonce` fields in the Vault payload and omitted them
when empty rather than sending null values.

---

**Date:** 2025-01-18
**Prompt:**
> "Add the `rewrap_data` tool. It should validate the ciphertext format before calling Vault
> (must start with vault:v), POST to transit/rewrap/:name, and return the new ciphertext.
> Explain in the return message what rewrap does vs decrypt+encrypt."

**Outcome:** `rewrap_data.go` with pre-call `validateCiphertext` check. Bob's return message
text clearly explains the no-plaintext-exposure guarantee, which became part of the demo narrative.

---

**Date:** 2025-01-19
**Prompt:**
> "Implement `generate_hmac` and `verify_hmac` as a pair. HMAC uses transit/hmac/:name/:algorithm,
> verify uses transit/verify/:name/:algorithm. Both need the same input auto-encoding logic as
> encrypt_data. verify_hmac should return a clear boolean verdict in human-readable form."

**Outcome:** `generate_hmac.go` and `verify_hmac.go`. Bob's verify return text: "HMAC verified
successfully: the input matches the HMAC" / "HMAC verification failed: the input does not match".
These messages are what Bob reads back in the demo.

---

## Testing

**Date:** 2025-01-20
**Prompt:** (using `.github/prompts/write-tool-tests.prompt.md`)
> "Write httptest-mocked tests for encrypt_data covering: happy path with payload assertion
> (confirm plaintext is base64-encoded before sending), bad base64 with plaintext_is_base64=true,
> missing name parameter, and Vault returning 500."

**Outcome:** `encrypt_data_test.go` with 5 cases. Bob correctly asserted that the plaintext
sent to the mock Vault server is the base64 encoding of the raw input, not the raw input itself.
This test caught a bug where the encoding was applied twice.

---

**Date:** 2025-01-20
**Prompt:** (using `.github/prompts/write-tool-tests.prompt.md`)
> "Write tests for decrypt_data. The malformed ciphertext test should verify the handler returns
> an error *without calling Vault* — demonstrate this by registering the endpoint and asserting
> it was never hit."

**Outcome:** `decrypt_data_test.go` with a `handlerCalled` flag in the mock handler. The test
confirms pre-call validation fires before the Vault endpoint is reached.

---

**Date:** 2025-01-21
**Prompt:**
> "Suggest edge cases for the e2e lifecycle test beyond the happy path. What could go wrong
> in the rotate → rewrap → verify sequence that unit tests won't catch?"

**Outcome:** Added these assertions to the e2e outline: (1) old ciphertext still decrypts
after rotation, (2) rewrapped ciphertext version matches latest_version from read_transit_key,
(3) HMAC verify returns false for a tampered input (one character changed). These became the
key assertions in `e2e/transit_e2e_test.go`.

---

## Documentation

**Date:** 2025-01-22
**Prompt:**
> "Draft docs/add-a-new-vault-engine.md from how we actually built the transit package. Make
> it a replicable playbook for a developer who wants to add SSH, Database, or LDAP engine support.
> Include the 8 steps from scaffolding through the agentic layer. Use create_transit_key.go as
> the worked example."

**Outcome:** `docs/add-a-new-vault-engine.md` — the full playbook with worked code example.
Bob organized the steps in the right order (scaffold helpers before tools, register after
tests pass, agentic layer last) and included the guardrails checklist.

---

**Date:** 2025-01-22
**Prompt:**
> "Write the README Transit Tools section. It should explain what Vault Transit is in two
> sentences, list the 13 tools in a table, show the 8-step demo workflow as example Bob prompts,
> and link to the detailed examples doc."

**Outcome:** The README Transit Tools section with tool table and quick-start example.

---

## Demo / Ops

**Date:** 2025-01-23
**Prompt:** (using `.github/prompts/transit-demo-runbook.prompt.md`)
> Run the transit-demo-runbook prompt against a running vault -dev instance.

**Outcome:** Clean 8-step live demo. Bob created the key, performed the encrypt/decrypt
round-trip, rotated the key, rewrapped the v1 ciphertext to v2, generated an HMAC, and
verified it — then intentionally tampered the input and showed `valid: false`. All narration
matched the expected flow. Total runtime: under 4 minutes.

---

**Date:** 2025-01-23
**Prompt:**
> "What Vault policy does the vault-mcp-server need to exercise all 13 Transit tools? Give me
> a minimal HCL policy that grants exactly the required capabilities."

**Outcome:** A minimal Vault policy with path `transit/*` capabilities `[create read update list]`.
Bob correctly excluded `delete` and `sudo` capabilities, keeping the policy least-privilege.
