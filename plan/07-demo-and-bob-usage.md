# 07 — Demo, Video & Bob Usage Log

> Covers the live demo flow, the 5-minute video structure, and the Bob-usage discipline that
> feeds the "Use of Bob across the SDLC" rubric criterion. Owned jointly by the orchestrator
> (demo execution) and WS-D (docs).

## Required deliverables (from the Bob-a-thon rules)

1. **Problem Statement & Solution** — `docs/problem-and-solution.md`.
2. **Working Code** — readable, executable GitHub repo (the fork + Transit package).
3. **Video Demo** — 5-minute video of the solution.
4. **Prompt & Bob Usage** — show the prompts and where Bob was used, with examples.
5. **Feedback on Bob usage** — all team members submit feedback before submission is enabled.

> Reminder: every team member must complete the Bob feedback form **before** project submission
> is unlocked. Do this early, not at the deadline.

## Live demo flow (run through Bob in VS Code)

Pre-reqs: `vault server -dev` running, `transit` enabled, server built, `.vscode/mcp.json`
loaded, Bob connected. Optionally switch Bob to the **Vault Transit Security** custom mode
(plan/05) to showcase the agentic guardrails.

| Step | Ask Bob to… | Expected result |
|------|-------------|-----------------|
| 1 | Create a Transit key named `customer-data` | Key created (type `aes256-gcm96`, non-exportable) |
| 2 | Read the key metadata | Type, version 1, capabilities shown |
| 3 | Encrypt the text `"the quick brown fox"` | `vault:v1:...` ciphertext returned |
| 4 | Decrypt that ciphertext | Original text recovered (round-trip) |
| 5 | Rotate the `customer-data` key | New version (v2) created |
| 6 | Rewrap the original (v1) ciphertext | New `vault:v2:...` ciphertext |
| 7 | Generate an HMAC for the text | `vault:v...` HMAC returned |
| 8 | Verify the HMAC | `valid: true`; tampered input → `valid: false` |

Narration emphasis: "Bob isn't just explaining Transit — it's **executing** real Vault crypto,
validating inputs, and recovering gracefully from errors."

### Backup plan
If Bob/VS Code has issues live, fall back to MCP Inspector:
```bash
npx @modelcontextprotocol/inspector ./vault-mcp-server
```
Drive the same tool sequence from the Inspector UI.

## 5-minute video structure

| Time | Segment | Show |
|------|---------|------|
| 0:00–0:30 | Problem | Apps need encryption; direct key management is risky/error-prone. |
| 0:30–1:15 | Solution | Vault Transit = encryption-as-a-service; we added Transit to Vault MCP. |
| 1:15–2:15 | Implementation | The `pkg/tools/transit/` package + how a tool maps to a Vault endpoint. |
| 2:15–4:00 | Live workflow | Bob runs create → encrypt → decrypt → rotate → rewrap (→ HMAC). |
| 4:00–4:40 | Testing & safety | Unit + e2e tests; validation, safe defaults, error handling. |
| 4:40–5:00 | Agentic best practices | Instructions, custom Bob mode, prompts, engine playbook. Close. |

> Spend the last 20s on the **agentic asset suite** — it's the rubric differentiator and easy to
> under-sell.

## Bob across the SDLC (what to demonstrate / log)

| SDLC phase | How Bob is used | Evidence to capture |
|------------|-----------------|---------------------|
| Planning | Compare Vault MCP enhancements; select Transit | prompts + decision notes |
| Design | Break feature into tools, params, returns, API mappings | this plan + tool registry |
| Implementation | Scaffold tools, validation, Vault request patterns | PRs + `add-transit-tool` prompt |
| Testing | Generate unit/e2e/validation tests, edge cases | `write-tool-tests` prompt + test files |
| Documentation | README, examples, troubleshooting, playbook | `docs/**` |
| Demo prep | Build the 5-min flow + runbook | `transit-demo-runbook` prompt |
| Ops | Enable engine, run server, inspect tools | setup commands |

## `docs/bob-usage-log.md` — template (everyone appends as they work)

```markdown
# Bob Usage Log

> Append an entry whenever Bob materially helped. Tag the SDLC phase. Paste the actual prompt
> and a one-line outcome. This is graded evidence — be specific.

## Planning
- [ ] **Prompt:** "Compare Transit, KV, PKI, and dynamic secrets for a Bob-a-thon scope by
  usefulness, complexity, and demo quality."
  **Outcome:** Selected Transit (security-relevant, self-contained, demo-friendly).

## Design
- [ ] **Prompt:** "Break Vault Transit support into MCP tools with names, parameters, return
  values, and Vault API paths."
  **Outcome:** 8 core + 5 stretch tools; see plan/04.

## Implementation
- [ ] **Prompt:** "Add a new Transit tool `rotate_transit_key` following plan/01 + plan/04..."
  **Outcome:** rotate_key.go + test, registered in tools.go.

## Testing
- [ ] **Prompt:** "Write httptest-mocked tests for encrypt_data covering bad base64 and a Vault error."
  **Outcome:** encrypt_data_test.go with 5 cases.

## Documentation
- [ ] **Prompt:** "Draft docs/add-a-new-vault-engine.md from how we built the transit package."
  **Outcome:** Reusable engine playbook.

## Demo / Ops
- [ ] **Prompt:** transit-demo-runbook.prompt.md
  **Outcome:** Clean 8-step live demo.
```

## Example Bob prompts (reusable, from the original plan)

- "Help us choose the best Vault MCP capability to implement for a Bob-a-thon project. Compare
  Transit, KV, PKI, and dynamic secrets based on usefulness, complexity, and demo quality."
- "Break down Vault Transit support into MCP tools with clear names, parameters, return values,
  and Vault API paths."
- "Help design a safe demo workflow for Vault Transit that shows create key, encrypt, decrypt,
  rotate, and rewrap."
- "Suggest unit and integration test cases for Vault Transit MCP tools, including base64
  validation, invalid ciphertext, missing keys, and Vault API errors."
- "Turn this implementation into a clear Bob-a-thon presentation explaining the problem,
  solution, Bob usage, and demo flow."
