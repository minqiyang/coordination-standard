# Claude-herdr Coordination Standard 0.11.0

A standard for running several AI coding agents on one project, with a Claude Code coordinator in a Herdr Tab. During development, workers edit and test as freely as they need. Before frozen code is first put to real use, QA checks it once, and so do fresh read-only reviewers when the lane requires them. Results come only from code that passed that check.

The owner watches only the coordinator in the main Herdr sidebar. Workers with a Tab run in a separate `workers` Herdr session that the sidebar does not show, and GPT seats run headless and read-only. The coordinator interrupts the owner only when all the work is done or when it and its workers are stuck.

You choose the models. `model_bindings.json` sets the model and effort of each role, and you can rebind any role. The defaults use Opus and GPT-6.1 Sol. A review seat can use any model that the `claude` or `codex` CLI can run, so review is not limited to Claude. Workers that write code stay on the `claude` CLI for now ([Choosing models](claude-herdr-coordination-standard/README.md#choosing-models)).

- Policy files: [`coordinator.md`](claude-herdr-coordination-standard/coordinator.md), [`model_bindings.json`](claude-herdr-coordination-standard/model_bindings.json)
- Start: [`claude-herdr-coordination-standard/README.md`](claude-herdr-coordination-standard/README.md)

![Owner focus](claude-herdr-coordination-standard/assets/owner_focus.svg)

![System architecture](claude-herdr-coordination-standard/assets/system_architecture.svg)

![Task flow](claude-herdr-coordination-standard/assets/workflow_lifecycle.svg)

The earlier Coordination Standard 0.13.0, for any coordinator agent, is kept on the [`coordination-standard-0.13`](https://github.com/minqiyang/coordination-standard/tree/coordination-standard-0.13) branch and is no longer maintained.

`archive/` (historical drafts, no authority) and `project-notes/` (project handoffs) are local only and not in git.
