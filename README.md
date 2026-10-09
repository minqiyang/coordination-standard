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

## 2. Claude-herdr Coordination Standard 0.10.0

For a Claude Code coordinator running in a Herdr Tab, derived from Coordination Standard 0.9.0. During development, workers edit and test as freely as they need. Before frozen code is first put to real use, QA checks it once, and so do fresh read-only reviewers when the lane requires them. Results come only from code that passed that check.

The owner watches only the coordinator in the main Herdr sidebar. Workers with a Tab run in a separate `workers` Herdr session that the sidebar does not show, and GPT seats run headless and read-only. The coordinator interrupts the owner only when all the work is done or when it and its workers are stuck.

- Policy files: [`coordinator.md`](claude-herdr-coordination-standard/coordinator.md), [`model_bindings.json`](claude-herdr-coordination-standard/model_bindings.json)
- Start: [`claude-herdr-coordination-standard/README.md`](claude-herdr-coordination-standard/README.md)

![Owner focus](claude-herdr-coordination-standard/assets/owner_focus.svg)

![System architecture](claude-herdr-coordination-standard/assets/system_architecture.svg)

![Task flow](claude-herdr-coordination-standard/assets/workflow_lifecycle.svg)
