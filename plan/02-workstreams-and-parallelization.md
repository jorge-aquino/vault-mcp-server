# 02 — Workstreams & Parallelization

> For the **orchestrator (main agent)**. This defines the parallel execution model, file
> ownership (so subagents never collide), the dependency graph, and copy-paste dispatch briefs.

## Orchestration model

```
                      ┌─────────────────────────────┐
                      │   Main agent = ORCHESTRATOR  │
                      │  owns shared files +         │
                      │  integration + build/test    │
                      └──────────────┬──────────────┘
        Phase 0 (foundation, blocks A/B/C) │
                      ┌───────────────┼───────────────┬───────────────┐
                      ▼               ▼               ▼               ▼
                  ┌───────┐       ┌───────┐       ┌───────┐       ┌───────┐
                  │ WS-A  │       │ WS-B  │       │ WS-C  │       │ WS-D  │  (from t0)
                  │ keys  │       │crypto │       │integ. │       │agentic│
                  └───┬───┘       └───┬───┘       └───┬───┘       └───────┘
                      └───────────────┴───────────────┘
                                      ▼
                                  ┌───────┐
                                  │ WS-E  │  (scaffold parallel; integrate after A/B/C)
                                  │ tests │
                                  └───────┘
                                      ▼
                      ┌─────────────────────────────┐
                      │  Phase 2: integrate, demo,   │
                      │  PR, finalize bob-usage-log  │
                      └─────────────────────────────┘
```

## Golden rule: file ownership = zero merge conflicts

| Owner | Files (exclusive write access) |
|-------|--------------------------------|
| **Orchestrator** | `pkg/tools/tools.go`, `pkg/tools/transit/transit_helpers.go`, `pkg/tools/transit/transit_test.go` (shared test helpers), repo setup, `.vscode/mcp.json` |
| **WS-A** | `pkg/tools/transit/{create_key,read_key,rotate_key,list_keys}.go` + matching `_test.go` |
| **WS-B** | `pkg/tools/transit/{encrypt_data,decrypt_data,rewrap_data}.go` + matching `_test.go` |
| **WS-C** | `pkg/tools/transit/{generate_hmac,verify_hmac,sign_data,verify_signature,hash_data,generate_random_bytes}.go` + matching `_test.go` |
| **WS-D** | `.github/**`, `docs/**`, `README.md` (Transit section), `SKILL.md` |
| **WS-E** | `e2e/transit_e2e_test.go`, `Makefile` (test targets), validation/regression test files |

> The only file multiple parties *want* to touch is `tools.go`. The orchestrator owns it and
> appends registrations as tool files are delivered — subagents just announce "tool ready."

## Dependency graph

| Workstream | Depends on | Can start | Blocks |
|------------|-----------|-----------|--------|
| Phase 0 Foundation | — | immediately | A, B, C, E-integration |
| WS-A Key mgmt | Phase 0 helpers + test helpers | after Phase 0 | E-integration |
| WS-B Encryption | Phase 0 | after Phase 0 | E-integration |
| WS-C Integrity | Phase 0 | after Phase 0 | E-integration |
| WS-D Agentic assets | — (docs only) | immediately (t0) | nothing |
| WS-E Testing/CI | scaffold: none; integration: A/B/C | scaffold at t0 | Phase 2 |

**Critical path:** Phase 0 → (A‖B‖C) → WS-E integration → Phase 2. WS-D runs entirely off the
critical path, so assign it to a subagent early to bank the rubric-heavy agentic assets.

## Subagent dispatch briefs (copy-paste)

Each brief is self-contained. Always attach: [01-system-architecture.md](01-system-architecture.md)
and [04-tool-specifications.md](04-tool-specifications.md). Remind every subagent to append the
prompts they used to `docs/bob-usage-log.md`.

### WS-A — Key Management & Lifecycle
```
You own the Transit key-management tools in pkg/tools/transit/.
Implement, following the exact patterns in plan/01-system-architecture.md and the specs in
plan/04-tool-specifications.md:
  - create_transit_key  (create_key.go)
  - read_transit_key    (read_key.go)
  - rotate_transit_key  (rotate_key.go)
  - list_transit_keys   (list_keys.go)   [stretch]
For each: a constructor returning server.ServerTool + a handler + a table-driven *_test.go
using the shared helpers in transit_test.go (mock Vault via httptest).
Use shared helpers from transit_helpers.go (transitPath, validateKeyName, extractString, etc.).
Safe defaults: type=aes256-gcm96, exportable=false, allow_plaintext_backup=false.
Annotations: read/list = ReadOnlyHint=true; create/rotate = DestructiveHint=false.
DO NOT edit tools.go or transit_helpers.go — announce each tool's constructor name when ready.
Run `gofmt` and `go test ./pkg/tools/transit/...`. Add the IBM copyright header to every file.
Log your prompts to docs/bob-usage-log.md under "Implementation".
```

### WS-B — Encryption Operations
```
You own the Transit encryption tools in pkg/tools/transit/:
  - encrypt_data  (encrypt_data.go)
  - decrypt_data  (decrypt_data.go)
  - rewrap_data   (rewrap_data.go)
Follow plan/01 + plan/04. Design decision (see plan/04): encrypt accepts raw `plaintext` and
base64-encodes it internally; add optional `plaintext_is_base64` flag. decrypt returns both the
raw base64 plaintext and (if valid UTF-8) the decoded text. Validate ciphertext `vault:v` prefix
before decrypt/rewrap using validateCiphertext() from transit_helpers.go.
Write *_test.go covering: happy path, round-trip, bad base64, malformed ciphertext, missing key.
DO NOT edit tools.go or transit_helpers.go. gofmt + go test. IBM headers. Log prompts.
```

### WS-C — Integrity & Crypto Utilities
```
You own the Transit integrity/crypto tools in pkg/tools/transit/:
  Core:    generate_hmac (generate_hmac.go), verify_hmac (verify_hmac.go)
  Stretch: sign_data, verify_signature, hash_data, generate_random_bytes
Follow plan/01 + plan/04. HMAC/sign algorithm defaults to sha2-256, passable via param.
verify_* returns the boolean `valid`. Inputs are base64; validate before calling Vault.
Write *_test.go for each (happy path + invalid input + verify-true/verify-false).
DO NOT edit tools.go or transit_helpers.go. gofmt + go test. IBM headers. Log prompts.
```

### WS-D — Agentic Assets & Documentation (start at t0)
```
You own all non-Go deliverables. Produce everything specified in plan/05-agentic-assets.md:
  - .github/copilot-instructions.md
  - .github/instructions/transit-tools.instructions.md  (applyTo: pkg/tools/transit/**)
  - .github/prompts/{add-transit-tool,write-tool-tests,transit-demo-runbook}.prompt.md
  - .github/agents/vault-transit.agent.md   (custom Bob mode)
  - .github/skills/vault-transit/SKILL.md
  - docs/{problem-and-solution,bob-usage-log,agentic-best-practices,demo-script,
          transit-tool-examples,testing-strategy,add-a-new-vault-engine}.md
  - README.md "Transit Tools" section
Keep tool names/params consistent with plan/04. This is the rubric differentiator — make the
"add-a-new-vault-engine" playbook genuinely reusable. Log your own prompts to bob-usage-log.md.
```

### WS-E — Testing, Integration & CI (scaffold at t0)
```
You own integration + CI. Per plan/06-testing-strategy.md:
  - e2e/transit_e2e_test.go: full lifecycle vs a real `vault server -dev` with transit enabled
    (create -> read -> encrypt -> decrypt round-trip -> rotate -> rewrap -> hmac -> verify).
  - Validation/error coverage and a regression check that kv/pki/sys tests still pass.
  - Makefile: add `test-transit` target; wire transit into `make test` / `make test-e2e`.
Scaffold the harness now; fill assertions as WS-A/B/C land tools. Coordinate tool names with
the orchestrator. gofmt + go vet. IBM headers. Log prompts.
```

## Orchestrator operating principles (context discipline)

> Goal: complete a large, multi-part build **without flooding the orchestrator's context**,
> while keeping each worker's context clean, minimal, and relevant. The orchestrator manages
> context as a scarce resource.

- **Decompose** the plan into narrow, self-contained, actionable tasks — one focused goal per
  worker. Use the [dispatch briefs](#subagent-dispatch-briefs-copy-paste) as the task units.
- **Minimal sufficient context:** give each worker ONLY what it needs — its relevant background,
  its owned files, constraints, and success criteria. No transcript, no unrelated history, no
  other workstreams' details.
- **Fresh worker per task:** dispatch each task to a fresh subagent (clean context). Do not reuse
  a worker across unrelated tasks.
- **You hold the state:** workers are stateless and must not depend on each other's memory.
  Cross-worker facts (tool names, param shapes, helper signatures) flow through YOU. You are the
  single source of truth and the only writer of shared files.
- **Keep only what matters in your own context:** decisions, the tool/file registry, and
  unresolved issues. Discard verbose worker logs once a result is validated.
- **Validate before moving on:** check every worker's output (build/test/spec compliance) before
  integrating. If a task is incomplete, incorrect, or creates new issues, refine the brief and
  reassign or fix it before continuing.
- **Sequence phases, parallelize within them:** proceed through phases/gates sequentially
  (0 → 1 → 2); inside Phase 1 the workers run in parallel. "Validate before moving on" applies at
  every integration point, not only at the end.

## Worker report-back contract

Every dispatched worker must **end its task** by reporting exactly these four things:

1. **Completed** — what the task achieved (1–2 lines).
2. **Changed** — exact files created/edited (and, for tools, the constructor names).
3. **Issues / risks / uncertainties** — anything unverified or assumption-laden.
4. **Follow-up required** — remaining work, or "none."

The orchestrator then: validates the report (inspect files, run `gofmt` + the package tests),
records the tool/file in the registry, and discards the verbose detail — keeping only the
decision and any open issue.

## Orchestrator runbook

1. **Phase 0:** complete [03-foundation-setup.md](03-foundation-setup.md). Confirm `make build`
   green and `transit_helpers.go` + `transit_test.go` compile.
2. **Dispatch** WS-A/B/C/E briefs; dispatch WS-D immediately.
3. **As tools land:** append their `AddTool` pairs to `tools.go`; run
   `go test ./pkg/tools/transit/...` after each merge.
4. **Integration gate:** `make build && make test`; then WS-E e2e against `vault -dev`.
5. **Phase 2:** `gofmt -l` / `go vet`, header check, MCP Inspector smoke, Bob-in-VS-Code dry
   run, record demo, finalize `docs/bob-usage-log.md`, open PR.

## Coordination conventions

- **Tool readiness signal:** subagent reports `pkg/tools/transit/<file>.go → constructor
  Transit<Name>(logger)`; orchestrator registers it.
- **No edits outside owned files.** If a shared helper is missing, request it from the
  orchestrator rather than editing `transit_helpers.go`.
- **Naming is contract:** tool names, params, and constructor names come from
  [04-tool-specifications.md](04-tool-specifications.md) — do not improvise.
- **Branch strategy (if humans co-work):** one branch per workstream off `feat/transit`,
  small PRs into `feat/transit`; orchestrator integrates.
