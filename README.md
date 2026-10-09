# Standards

Two peer coordination standards for running several AI coding agents on one project with Herdr. A coordinator session loads exactly one of them.

`archive/` (historical drafts, no authority) and `project-notes/` (project handoffs) are local only and not in git.

## 1. Coordination Standard 0.13.0

For any coordinator agent. One coordinator hands out task cards, reads the workers' reports, and decides what gets accepted, with review depth set by a lane for each card.

- Policy files: [`coordinator.md`](coordination-standard/coordinator.md), [`routing_table.json`](coordination-standard/routing_table.json), [`model_bindings.json`](coordination-standard/model_bindings.json)
- Start: [`coordination-standard/README.md`](coordination-standard/README.md)

![System architecture](coordination-standard/assets/system_architecture.svg)

![Workflow lifecycle](coordination-standard/assets/workflow_lifecycle.svg)

![Auxiliary components](coordination-standard/assets/auxiliary_components.svg)

## 2. Claude-herdr Coordination Standard 0.9.0

For a Claude Code coordinator running in a Herdr Tab, derived from Coordination Standard 0.9.0. The owner watches only the coordinator in the main Herdr sidebar. The coordinator interrupts the owner only when all the work is done or when it and its workers are stuck. Each producer, integrator, Claude seat, and adjudicator gets a new Tab and git worktree in a separate `workers` Herdr session that the sidebar does not show. GPT seats run headless and read-only as background `codex exec` tasks, each in its own detached worktree.

One worker owns a whole dev loop. It edits, runs, and reads results in its worktree, and stops to report at the checkpoints its card names or when it is blocked. The lane's gate runs once, at the first real use of a frozen commit, such as a run of record, a merge, or input to another card. Runs of record use only accepted code, and numbers from dev runs never feed a result.

- Policy files: [`coordinator.md`](claude-herdr-coordination-standard/coordinator.md), [`model_bindings.json`](claude-herdr-coordination-standard/model_bindings.json)
- Start: [`claude-herdr-coordination-standard/README.md`](claude-herdr-coordination-standard/README.md)

![Owner focus](claude-herdr-coordination-standard/assets/owner_focus.svg)

![System architecture](claude-herdr-coordination-standard/assets/system_architecture.svg)

![Workflow lifecycle](claude-herdr-coordination-standard/assets/workflow_lifecycle.svg)

## Versioning

Both standards use semantic versioning: `MAJOR.MINOR.PATCH`, where each part is an integer, so 0.10.0 follows 0.9.0. Each stays at 0.x until the owner declares it stable as 1.0.0.

- MINOR: a change to what the coordinator or agents do (rules, routes, gates, fixed prompts).
- PATCH: no change in behavior (wording, README, diagrams, or a model ID swap in the bindings).
- Every change to a policy file gets a new number. The card title, README, start and switch prompts, and JSON version fields always match.

Coordination Standard 9.0 was renamed 0.9.0; earlier releases were numbered V1 to V9.0.
