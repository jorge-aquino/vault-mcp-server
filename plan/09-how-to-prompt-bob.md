# 09 — How to Prompt Bob to Execute This Plan

> Practical guide for driving the build with Bob. Owned by the **orchestrator**. Read this
> alongside [02-workstreams-and-parallelization.md](02-workstreams-and-parallelization.md)
> (the runbook) and [08-verification-risks-timeline.md](08-verification-risks-timeline.md)
> (the definition of done).

## TL;DR

- **Do NOT use one giant "build everything" prompt.** Phase 0 is a hard dependency — WS-A/B/C
  can't compile without `transit_helpers.go` + `transit_test.go`. A mega-prompt makes Bob spawn
  subagents before the foundation exists, causing `tools.go` collisions and "I think it's done"
  failures with no real verification.
- **Use one master *orchestrator* prompt that drives the plan in gated phases and dispatches
  subagents itself.** You keep a human checkpoint at each gate; completeness is enforced against
  the [plan/08 acceptance checklist](08-verification-risks-timeline.md#acceptance-criteria-all-must-be-true-to-submit).

```
Prompt 1  →  Orchestrator kickoff + Phase 0  →  STOP at exit gate, report
Prompt 2  →  Dispatch WS-A/B/C/D/E in parallel as subagents  →  integrate, report
Prompt 3  →  Run plan/08 acceptance gate  →  fix gaps  →  declare done
```

---

## Prompt 1 — Orchestrator kickoff (copy-paste)

Attach the whole `plan/` folder, switch Bob to Agent mode, then send:

```
You are the ORCHESTRATOR for this project. Read the entire plan/ folder before acting,
especially plan/02-workstreams-and-parallelization.md (your runbook) and
plan/08-verification-risks-timeline.md (your definition of done).

Operating rules (non-negotiable):
- Follow the file-ownership table in plan/02 exactly. ONLY you may edit
  pkg/tools/tools.go, pkg/tools/transit/transit_helpers.go, and transit_test.go.
- Use the naming contract in plan/04 verbatim — do not improvise tool, param, or
  constructor names.
- Work in gated phases. Do NOT start a phase until the previous gate passes. Pause and
  report to me at each gate; wait for my "go" before the next phase.
- Manage context as a scarce resource (see plan/02 "Orchestrator operating principles").
  Dispatch each task to a FRESH worker with ONLY the files, constraints, and success criteria
  it needs — no transcript, no unrelated history, no other workstream's details. Keep only
  decisions, the tool/file registry, and open issues in your own context; discard verbose logs.
- You are the single source of truth for cross-worker facts (tool names, param shapes, helper
  signatures). Workers are stateless and must not depend on each other's memory.
- Validate every worker's output (build/test/spec compliance) before moving on. If a result is
  incomplete, wrong, or creates new issues, refine the brief and reassign before continuing.

Do Phase 0 now (plan/03-foundation-setup.md):
1. Confirm the fork/clone, Vault dev server, and transit engine are set up (give me the
   exact commands if anything is missing).
2. Create the transit package scaffolding: transit_helpers.go and transit_test.go with the
   full helper surface listed in plan/03, plus the empty registration hook in tools.go.
3. Run `make build` and confirm the package compiles.`

STOP at the Phase 0 exit criteria. Report: what you created, build status, and the list of
subagent briefs you will dispatch next. Do not write any tool files yet.
```

Why it works: forces context loading, locks the ownership + naming contracts up front (the two
things that break parallel work), and stops at a verifiable gate instead of charging ahead.

---

## Prompt 2 — Dispatch the parallel workstreams

After Phase 0 is green:

```
Phase 0 is approved. Now dispatch the workstreams as parallel subagents using the
copy-paste briefs in plan/02 ("Subagent dispatch briefs"):
- WS-A (key mgmt), WS-B (encryption), WS-C (integrity), WS-D (agentic assets), WS-E (tests).
Attach plan/01 and plan/04 to each subagent. Start WS-D immediately (it's off the critical
path). Each subagent owns ONLY its files and must NOT touch tools.go or transit_helpers.go.

Give each worker only its own brief + plan/01 + plan/04 — not this whole plan or any other
workstream's context. Require each worker to END by reporting (the plan/02 report-back
contract): (1) what it completed, (2) files/constructors changed, (3) issues/risks/
uncertainties, (4) follow-up required.

As each subagent reports a tool ready, VALIDATE its report (inspect files, run gofmt +
`go test ./pkg/tools/transit/...`) BEFORE you register its AddTool pair in tools.go. If a
worker's output is incomplete or wrong, refine its brief and reassign before continuing.
After all land, run `make build && make test`. Keep only the tool/file registry and any open
issues in your context — discard verbose worker logs. Report a table of every tool, its file,
and its test status. Then stop.
```

The subagents don't need bespoke prompts — the briefs in
[plan/02](02-workstreams-and-parallelization.md#subagent-dispatch-briefs-copy-paste) are already
self-contained and Bob will paste them in.

---

## Prompt 3 — Completeness / verification gate

This is the "make sure all work is complete" step.

```
Run the full acceptance gate from plan/08-verification-risks-timeline.md. Go through the
checklist literally and mark each item pass/fail with evidence:
- functional (all 8 core tools + selected stretch callable),
- quality (make build, make test, e2e vs vault -dev, gofmt -l, go vet, IBM headers, no secrets logged),
- agentic assets present (instructions, prompts, custom mode, SKILL, engine playbook),
- submission deliverables (problem-and-solution.md, README Transit section, bob-usage-log.md filled).

For every FAIL, fix it (or re-dispatch the owning subagent), then re-run that check. Do not
tell me it's done until every box in plan/08 passes.

End with a FINAL SUMMARY:
- What was completed
- What changed (files/tools)
- What was validated (which checks ran green)
- Anything missed, skipped, blocked, or still requiring attention
- Recommended next steps
```

---

## How to guarantee nothing is missed

- **The [plan/08 acceptance checklist](08-verification-risks-timeline.md#acceptance-criteria-all-must-be-true-to-submit)
  is the completeness contract** — Prompt 3 forces Bob to grade itself against it instead of
  self-declaring "done."
- **Gates prevent the classic failure** where a subagent invents a tool name and `tools.go`
  won't compile. Ownership + naming are locked in Prompt 1.
- **Start WS-D early** to bank the rubric-heavy agentic assets even if engineering runs long.
- **One non-Bob safety net:** run the
  [plan/08 verification commands](08-verification-risks-timeline.md#verification-checklist-orchestrator-runs-before-demo)
  (`make build`, `make test`, `make test-transit-e2e`, MCP Inspector) yourself before recording
  the demo — never trust "tests pass" without seeing green output.

## One-prompt alternative (faster, riskier)

You *can* paste Prompts 1–3 together and tell Bob to run all phases sequentially with the same
gates. It's faster but loses the human checkpoint: if Phase 0 is subtly wrong, all five
workstreams inherit the error before you can catch it. For a hackathon where a broken build at
demo time is fatal, the 3-prompt gated approach is the safer bet.

## Quick reference

| Step | Prompt | Gate before moving on |
|------|--------|-----------------------|
| 1 | Orchestrator kickoff + Phase 0 | `make build` green; helpers + test helpers compile |
| 2 | Dispatch WS-A/B/C/D/E as subagents | `make build && make test` green; all tools registered |
| 3 | Run plan/08 acceptance gate | every checklist box passes with evidence |
