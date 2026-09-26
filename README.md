# Coordination Standard

**Version `8.6`**

A working standard for running several AI coding agents on one project. **Herdr** hosts every session in its own terminal Tab, and one agent session acts as the coordinator: it hands out tasks, reads the workers' reports, and decides what gets accepted. The coordinator does not write the code itself, and any capable agent can play that role.

The policy lives in three files, and only these three:

| File | What it holds | Share it? |
|---|---|---|
| [`coordinator.md`](coordinator.md) | The operating card: who may write where, how risky work is reviewed, how reports are handed back, and when work may be merged or published | Yes, portable |
| [`routing_table.json`](routing_table.json) | The roles (routes), when each one is used, which review seats each lane needs, the merge gate, and the fixed prompts. No model names | Yes, portable |
| [`model_bindings.json`](model_bindings.json) | Your subscriptions, and which model, harness, and reasoning effort each role uses right now | Personal; replace with your own |

When a new model comes out, only `model_bindings.json` changes. To give the standard to someone else, send the first two files and let them write their own bindings.

This README and its diagrams only explain those files. If they ever disagree, the policy files win.

## How it fits together

![System architecture](assets/system_architecture.svg)

Each delegated task gets a card naming one route and its own report file, and any directory being changed has exactly one writer at a time. The coordinator waits, reads the report, checks the actual output, and only then moves on.

![Workflow lifecycle](assets/workflow_lifecycle.svg)

## Lanes

Every card gets one lane, based on what a mistake would cost:

| Lane | Typical work | What it takes to accept |
|---|---|---|
| ROUTINE | Routine, local, easily reversed changes | QA passes and the coordinator checks the output |
| ELEVATED | Important behavior or correctness changes | QA passes plus one fresh, independent reviewer |
| CRITICAL | Mistakes that could corrupt state, cross a trust boundary, or can't be undone | QA passes plus two fresh, independent reviewers on different models, at higher reasoning effort |

Some work is also marked **structural**, such as project-defining architecture, security boundaries, or shared contracts. Structural work needs an accepted binding plan first and then the CRITICAL gate. The exact criteria are in [section 2 of the card](coordinator.md#2-lanes-and-gates).

## Getting started

Open a Herdr Tab for your coordinator agent and point it at the three policy files where they are. Don't copy them, so there is only ever one live version:

```text
Read <standard-dir>/coordinator.md, <standard-dir>/routing_table.json, and
<standard-dir>/model_bindings.json as the coordinator workflow policy, within
the user's authorization and project constraints.
Declared version: 8.6. Do not load archive/. Follow the card.
```

## Optional tools

![Auxiliary components](assets/auxiliary_components.svg)

**Contract First** ([`skills/contract-first/SKILL.md`](skills/contract-first/SKILL.md), installed to `~/.agents/skills/contract-first/SKILL.md`) is a short skill. Before changing code or a deliverable, the agent pins down four things: who owns the interface, what must stay true, what it's allowed to do and where, and which checks prove it worked.

**Artifact Guard** ([`extensions/artifact-guard/index.ts`](extensions/artifact-guard/index.ts), installed to `~/.pi/agent/extensions/artifact-guard/index.ts`) is a Pi extension that catches accidental drift, such as writing to the wrong directory or skipping QA. It is a tripwire, not a sandbox.

- `contract_check` records the allowed write roots, frozen inputs, required outputs, and QA commands.
- `artifact_finalize` checks the outputs and runs the QA commands you pass it. Its PASS is not project acceptance.
- If a `.artifact-guard.json` file exists in the current directory or any parent up to the repository root, the guard runs in ENFORCE mode and blocks edits until a contract exists. Otherwise it runs in ADVISORY mode. In both modes, once a contract exists, writes outside its roots are blocked.
