# Vault MCP Transit Security Extension — Development Plan

> Bob-a-thon 2026 (IBM Austin) submission. This `plan/` folder is the complete,
> implementation-ready specification for extending **`hashicorp/vault-mcp-server`** with a
> HashiCorp Vault **Transit** (encryption-as-a-service) capability suite, so **Bob** can
> guide *and execute* real Vault-backed crypto workflows through MCP.

**Status:** Planning complete — ready for implementation. _No code has been written yet._

---

## How to use these documents

These docs are designed for an **orchestrator + parallel subagents** model:

- The **main agent (orchestrator)** reads [02-workstreams-and-parallelization.md](02-workstreams-and-parallelization.md),
  completes **Phase 0 foundation** ([03-foundation-setup.md](03-foundation-setup.md)), then
  dispatches the five workstreams to subagents and owns integration.
- Each **subagent** is handed exactly one workstream brief plus the shared references
  ([01-system-architecture.md](01-system-architecture.md) and
  [04-tool-specifications.md](04-tool-specifications.md)).
- Everyone logs the prompts they used to `docs/bob-usage-log.md` (template in
  [07-demo-and-bob-usage.md](07-demo-and-bob-usage.md)) — this directly feeds the judging rubric.

## Document index

| # | Document | Audience | Purpose |
|---|----------|----------|---------|
| — | [README.md](README.md) | All | This index + orchestration model |
| 00 | [00-overview-and-win-strategy.md](00-overview-and-win-strategy.md) | All | Problem, solution, rubric alignment, win strategy |
| 01 | [01-system-architecture.md](01-system-architecture.md) | All subagents | Existing repo deep-dive, code patterns, Transit API map |
| 02 | [02-workstreams-and-parallelization.md](02-workstreams-and-parallelization.md) | Orchestrator | Dependency graph, file ownership, subagent dispatch briefs |
| 03 | [03-foundation-setup.md](03-foundation-setup.md) | Orchestrator | Phase 0: fork, Vault dev, helpers, registration, `.vscode/mcp.json` |
| 04 | [04-tool-specifications.md](04-tool-specifications.md) | WS-A/B/C | Full tool specs + Go code skeletons + unit test pattern |
| 05 | [05-agentic-assets.md](05-agentic-assets.md) | WS-D | Instructions, prompts, custom Bob mode, SKILL, engine playbook |
| 06 | [06-testing-strategy.md](06-testing-strategy.md) | WS-E | Unit/integration/validation/regression strategy + CI |
| 07 | [07-demo-and-bob-usage.md](07-demo-and-bob-usage.md) | Orchestrator/WS-D | Demo script, 5-min video, Bob-usage-log + SDLC mapping |
| 08 | [08-verification-risks-timeline.md](08-verification-risks-timeline.md) | Orchestrator | Acceptance criteria, verification, risks, timeline, deliverables |
| 09 | [09-how-to-prompt-bob.md](09-how-to-prompt-bob.md) | Orchestrator | How to prompt Bob to execute the plan (gated 3-prompt flow) |

## At-a-glance

- **Base:** fork of `hashicorp/vault-mcp-server` (Go 1.24+, `mark3labs/mcp-go`).
- **New package:** `pkg/tools/transit/` — 8 core tools + select stretch tools.
- **Core tools:** `create_transit_key`, `read_transit_key`, `rotate_transit_key`,
  `encrypt_data`, `decrypt_data`, `rewrap_data`, `generate_hmac`, `verify_hmac`.
- **Stretch tools:** `list_transit_keys`, `sign_data`, `verify_signature`, `hash_data`,
  `generate_random_bytes`.
- **Differentiator:** a full **reusable agentic-asset suite** (instructions / prompts /
  custom Bob mode / SKILL / "add a new Vault engine" playbook).
- **Demo client:** Bob in VS Code via `.vscode/mcp.json` (MCP Inspector as fallback).
- **Parallelism:** 5 workstreams, disjoint file ownership, orchestrator owns shared files.

## Quick links to the judging rubric

The three official criteria and where we address each:

1. **Use of Bob across SDLC** → [07-demo-and-bob-usage.md](07-demo-and-bob-usage.md)
2. **Complexity of enhancement** → [01-system-architecture.md](01-system-architecture.md) + [05-agentic-assets.md](05-agentic-assets.md) (engine playbook)
3. **Agentic best practices (reusable/replicable)** → [05-agentic-assets.md](05-agentic-assets.md)
