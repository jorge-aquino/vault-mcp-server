# 08 — Verification, Risks, Timeline & Deliverables

> Owned by the **orchestrator**. The acceptance gate for "done," the risk register, an
> indicative timeline, and a deliverables-to-rubric traceability map.

## Acceptance criteria (all must be true to submit)

### Functional
- [ ] All **8 core** tools implemented and callable through Bob: `create_transit_key`,
  `read_transit_key`, `rotate_transit_key`, `encrypt_data`, `decrypt_data`, `rewrap_data`,
  `generate_hmac`, `verify_hmac`.
- [ ] Selected **stretch** tools implemented: `list_transit_keys`, `sign_data`,
  `verify_signature`, `hash_data`, `generate_random_bytes`.
- [ ] Full workflow works end to end: create → read → encrypt → decrypt → rotate → rewrap →
  hmac → verify.
- [ ] `mount` defaults to `transit`; explicit mounts honored.

### Quality
- [ ] `make build` succeeds; `make test` green (unit + validation + regression).
- [ ] e2e lifecycle passes vs real `vault server -dev`.
- [ ] `gofmt -l ./pkg/tools/transit` prints nothing; `go vet ./...` clean.
- [ ] IBM copyright + SPDX header on every new file.
- [ ] No secrets/tokens/plaintext/key-material logged.

### Agentic assets (differentiator)
- [ ] `.github/copilot-instructions.md` + `transit-tools.instructions.md` present.
- [ ] Prompt library (3 prompts) present and working.
- [ ] `vault-transit.agent.md` custom mode works in Bob.
- [ ] `SKILL.md` present.
- [ ] `docs/add-a-new-vault-engine.md` playbook present and genuinely reusable.

### Submission deliverables
- [ ] `docs/problem-and-solution.md` written.
- [ ] Repo readable & executable (README updated with Transit section + run steps).
- [ ] 5-minute video recorded.
- [ ] `docs/bob-usage-log.md` populated across all SDLC phases.
- [ ] **All team members submitted Bob feedback** (unlocks submission).

## Verification checklist (orchestrator runs before demo)

```bash
# 1. Build & static
make build
gofmt -l ./pkg/tools/transit          # expect empty
go vet ./...

# 2. Unit + regression
make test

# 3. Integration (Vault dev must be running + transit enabled)
export VAULT_ADDR=http://127.0.0.1:8200 VAULT_TOKEN=root
make test-transit-e2e

# 4. Tool discovery smoke
npx @modelcontextprotocol/inspector ./vault-mcp-server   # list all transit tools

# 5. Bob dry run: execute the 8-step demo from plan/07
```

## Risk register

| Risk | Prob. | Impact | Mitigation |
|------|:-----:|:------:|-----------|
| Base64 encoding errors | Med | Low | Accept raw text + auto-encode; `validateBase64`; clear messages. |
| Ciphertext format issues | Low | Low | `validateCiphertext` (`vault:v` prefix) before decrypt/rewrap. |
| Key version confusion | Low | Med | Expose metadata via `read_transit_key`; show rotate/rewrap in demo. |
| Shared-file merge conflicts | Med | Med | Orchestrator solely owns `tools.go`/`transit_helpers.go`; subagents own disjoint files. |
| Test-helper duplication across packages | Med | Low | Phase 0 ships `transit_test.go`; subagents reuse, never import cross-package test code. |
| Vault API write verb (PUT vs POST) in mocks | Med | Low | Document in plan/06; assert PUT in mocks. |
| Scope creep into stretch tools | Med | Med | Lock the 8 core first; stretch only after core is green. |
| Insufficient Bob framing | Med | High | Maintain `bob-usage-log.md` continuously; dedicate demo time to it. |
| Live demo failure | Low | High | MCP Inspector fallback; pre-recorded backup clip. |
| Team Bob feedback not submitted | Med | High | Complete feedback on day 1, not at deadline. |

## Indicative timeline (single intensive day, parallelized)

| Phase | Work | Owner | Rough effort |
|-------|------|-------|--------------|
| 0 | Fork, Vault dev, helpers, `mcp.json`, registration stub | Orchestrator | 1–1.5h (blocks A/B/C) |
| 1a | Core encryption + key tools (WS-A, WS-B) | 2 subagents | 3–4h parallel |
| 1b | Integrity tools (WS-C) | 1 subagent | 2–3h parallel |
| 1c | Agentic assets + docs (WS-D) | 1 subagent | 3–4h parallel (off critical path) |
| 1d | Test harness + e2e (WS-E) | 1 subagent | 2–3h (integration after 1a/1b) |
| 2 | Integration, polish, demo, video, PR | Orchestrator + all | 2–3h |

> WS-D runs entirely off the critical path — start it at t0 so the rubric-heavy assets are
> banked early even if engineering runs long.

## Deliverables → rubric traceability

| Deliverable | Rubric criterion it serves |
|-------------|----------------------------|
| `pkg/tools/transit/` (extending a real server) | Complexity of enhancement |
| `docs/add-a-new-vault-engine.md` | Complexity + Agentic best practices |
| `.github/{instructions,prompts,agents,skills}` | Agentic best practices (reusable) |
| `docs/bob-usage-log.md` | Use of Bob across SDLC |
| Unit + e2e tests | Complexity (depth) + working code |
| 5-min video + demo | Working code + Bob-in-action |
| Custom Bob mode (`vault-transit.agent.md`) | Agentic best practices |

## Success metrics

| Category | Metric |
|----------|--------|
| Implementation | All 8 core tools callable through Vault MCP. |
| Workflow | End-to-end create→encrypt→decrypt→rotate→rewrap→hmac→verify works. |
| Testing | Unit + integration cover success and failure paths. |
| Documentation | README, tool examples, troubleshooting, playbook, demo script. |
| Bob usage | Documented across planning, design, coding, testing, docs, demo. |
| Agentic assets | Reusable instructions/prompts/mode/skill/playbook shipped. |
| Demo | 5-min video clearly shows problem, solution, implementation, live workflow. |

## Open decisions to confirm before/at kickoff

1. **`enable_transit` tool vs `create_mount` extension vs CLI-only** — recommended: CLI in setup
   now; add `create_mount` "transit" support as a stretch to show system mastery.
2. **Upstream PR** — opening a PR to `hashicorp/vault-mcp-server` would strengthen the
   "adoption/replicability" rubric point. Decide based on time remaining.
3. **Key type breadth** — default `aes256-gcm96` for symmetric; only add an asymmetric key in the
   demo if showing `sign_data`/`verify_signature`.
