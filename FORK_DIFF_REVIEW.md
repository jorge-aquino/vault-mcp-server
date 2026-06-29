# Vault MCP Server Fork Diff Review

## Comparison Scope

Compared this fork against the official HashiCorp `vault-mcp-server` upstream `main`.

- Upstream base: `e1e2e339ca9f0c316e55ca2e22873a3f020618fd`
- Fork HEAD: `4673c8098b049842e317fbc1cdd9cad54dd1eb1d`
- Changed files: 75
- Insertions: 10,618
- Deletions: 23

Major additions include Transit engine tools, PKI deletion/revocation tools, expanded PKI tests, Transit docs, planning documents, and agentic/Bob assets.

## Code Review Findings

### 1. High: `delete_transit_key` bypasses Vault's deletion safety gate

File: `pkg/tools/transit/delete_key.go`

`delete_transit_key` auto-sets `deletion_allowed=true` before deleting the key. Vault intentionally requires a separate configuration step before irreversible key deletion. Collapsing both operations into one MCP tool is risky, especially when the caller may be an LLM.

Recommendation:

- Do not auto-enable deletion in `delete_transit_key`.
- Require users to call `update_transit_key(deletion_allowed=true)` separately.
- Alternatively, require an explicit confirmation parameter and document the irreversible behavior clearly.

### 2. Medium/High: `update_transit_key` likely sends wrong types for version config

File: `pkg/tools/transit/update_key.go`

`min_decryption_version` and `min_encryption_version` are exposed as strings and sent to Vault as strings. Vault Transit key config expects numeric version values.

Recommendation:

- Change schema fields to `mcp.WithNumber`.
- Parse and validate integer values.
- Send integer values to Vault.
- Add handler tests for these fields.

### 3. Medium: PKI default `mount` schema does not match handler behavior

Files:

- `pkg/tools/pki/delete_pki_issuer.go`
- `pkg/tools/pki/revoke_pki_certificate.go`
- `pkg/utils/utils.go`

The new PKI tools declare `mount` with default `pki`, but handlers call `utils.ExtractMountPath`, which errors when `mount` is omitted. Clients that trust the MCP schema default can receive an unexpected validation error.

Recommendation:

- Make `ExtractMountPath` accept a default value, or add PKI-specific default handling before calling it.
- Apply this consistently across PKI tools that declare `mcp.DefaultString("pki")`.

### 4. Medium: Transit e2e test bypasses the MCP server

File: `e2e/transit_e2e_test.go`

The Transit e2e test uses the Vault API client directly. This validates Vault endpoint behavior, but not MCP registration, handler parsing, tool-call behavior, or response formatting.

Recommendation:

- Keep the Vault API test if useful, but do not treat it as MCP e2e coverage.
- Add at least one MCP-level e2e/smoke test that starts the MCP server and performs `tools/call`.
- Cover representative tools such as `create_transit_key`, `encrypt_data`, `decrypt_data`, and `rotate_transit_key`.

### 5. Medium: PR hygiene risk from local/agentic artifacts

Added artifacts include:

- `.bob/custom_modes.yaml`
- `.github/agents/*`
- `.github/prompts/*`
- `.github/skills/*`
- `plan/*`
- `plan/vault_mcp_transit_bobathon_plan_polished.pdf`

These look like hackathon/planning/agent workflow assets rather than upstream product code. They significantly increase review noise and may not belong in a HashiCorp upstream PR.

Recommendation:

- Remove these from the main feature PR.
- If they are valuable, submit them separately as internal enablement/demo material.
- Remove "Bob" references from product documentation unless HashiCorp explicitly wants that branding.

### 6. Low/Medium: Docs list 13 Transit tools, but code registers more

Files:

- `README.md`
- `pkg/tools/tools.go`

The README says there are 13 Transit tools, but the code also registers `enable_transit`, `delete_transit_key`, and `update_transit_key`.

Recommendation:

- Align the README with the actual registered tools.
- Either include all Transit-related tools in the table or clearly separate engine setup, key management, and crypto operations.

## What This Fork Covers

### Core MCP Server

- Stdio transport.
- Streamable HTTP transport.
- Vault configuration via environment variables, headers, query/context values depending on transport.
- Session-based Vault client management.
- Middleware for HTTP logging, Vault context, CORS, and rate limiting.

### System and Mount Management

- List mounts.
- Create mounts.
- Delete mounts.
- Enable Transit mount.

### KV Secrets

- List secrets.
- Read secrets.
- Write secrets.
- Delete secrets.
- Delete a single key from a secret.
- KV v1 and KV v2 path handling in key areas.

### PKI

- Enable PKI.
- Create PKI issuers.
- List PKI issuers.
- Read PKI issuers.
- Delete PKI issuers.
- Create PKI roles.
- List PKI roles.
- Read PKI roles.
- Delete PKI roles.
- Issue PKI certificates.
- Revoke PKI certificates.

### Transit

- Enable Transit.
- Create Transit keys.
- Read Transit key metadata.
- List Transit keys.
- Rotate Transit keys.
- Update Transit key configuration.
- Delete Transit keys.
- Encrypt data.
- Decrypt data.
- Rewrap ciphertext.
- Generate HMACs.
- Verify HMACs.
- Sign data.
- Verify signatures.
- Hash data.
- Generate random bytes.

## What May Be Missing

### Transit

- Batch encrypt/decrypt/rewrap/HMAC/sign operations.
- Data key generation.
- Key export.
- Key import.
- Key backup and restore.
- Trimming old key versions.
- More complete key type support.
- More explicit support for convergent encryption workflows.
- Tests for `update_transit_key` and `delete_transit_key`.

### KV

- KV v2 metadata operations.
- KV v2 undelete.
- KV v2 destroy.
- Version rollback or version-specific reads.
- Metadata patch/list workflows.

### PKI

- Import existing issuers.
- More complete intermediate CA workflows.
- CRL configuration.
- Issuing URLs and AIA URL management.
- PKI tidy operations.
- More certificate request options such as SANs, IP SANs, URI SANs, email SANs, and custom TTL constraints.

### Vault Administration

- Policy management.
- Token management.
- Identity/group/entity management.
- Auth method management such as AppRole, Kubernetes, JWT/OIDC, LDAP, and userpass.
- Audit device management.

### MCP and Output Quality

- Consistent structured JSON outputs across tools.
- MCP-level e2e tests.
- More explicit tool annotations for dangerous operations.
- Consistent default handling between MCP schema and handlers.

## PR Readiness Recommendation

Do not submit the current full diff as one upstream PR. It is too broad and includes product code, tests, docs, planning files, local agent assets, and a binary PDF.

Recommended PR split:

1. Transit core tools PR
   - Transit implementation files.
   - Tool registration.
   - Focused unit tests.
   - MCP-level smoke/e2e test.
   - Minimal product documentation.

2. PKI improvements PR
   - Delete issuer.
   - Revoke certificate.
   - Default handling fixes.
   - PKI tests.

3. Documentation PR
   - User-facing Transit documentation.
   - Remove Bob/hackathon-specific wording.
   - Keep examples aligned with actual output.

4. Internal agent/demo assets
   - Keep out of upstream product PR unless requested.
   - If needed, publish separately as internal enablement material.

## Validation Performed

- `go test ./...` passed.
- MCP smoke test passed:
  - Server returned `vault-mcp-server`.
  - Version returned `0.2.1`.
  - `tools/list` returned 34 tools.
- Worktree was clean after review.

## Overall Assessment

The fork substantially expands the Vault MCP server into KV, PKI, and Transit coverage that is useful for real Vault workflows. The Transit implementation is broad and mostly follows Vault API path conventions. The PKI additions improve lifecycle completeness.

Before a professional upstream PR, the main work is not adding more code. The priority should be reducing scope, removing non-product artifacts, tightening dangerous operation behavior, aligning schemas with handlers, and adding MCP-level integration coverage.
