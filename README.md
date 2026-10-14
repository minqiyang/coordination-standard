# Herdr Coordination Standard 0.12.3

A standard for running several AI coding agents on one project. One coordinator agent in a Herdr Tab runs the work. During development, workers edit and test as freely as they need. Before frozen code is first put to real use, QA checks it once, and so do fresh read-only reviewers when the lane requires them. Results come only from code that passed that check.

Every role is a slot that you fill with the agent CLI and the model you choose. The coordinator can be any agent CLI that can wait on background tasks, such as Claude Code. Each worker runs on an agent CLI that Herdr starts, such as Claude Code, Codex, Kimi Code, Droid, or Devin, with any model that CLI runs. The rules name roles, never models.

![Agents and models](claude-herdr-coordination-standard/assets/agents_and_models.svg)

You watch only the coordinator in the main Herdr sidebar. Workers run in a separate `workers` Herdr session that the sidebar does not show, or headless. The coordinator interrupts you only when all the work is done or when it and its workers are stuck.

![System architecture](claude-herdr-coordination-standard/assets/system_architecture.svg)

![Task flow](claude-herdr-coordination-standard/assets/workflow_lifecycle.svg)

- Policy files: [`coordinator.md`](claude-herdr-coordination-standard/coordinator.md), [`model_bindings.json`](claude-herdr-coordination-standard/model_bindings.json)
- Start: [`claude-herdr-coordination-standard/README.md`](claude-herdr-coordination-standard/README.md)
- Optional guard: [`guard/`](guard/), a hook for a Claude Code coordinator that blocks three repeated mistakes (seat without read-only sandbox, merge of an unaccepted SHA, seat before QA passes)

The earlier Coordination Standard 0.13.0 is kept on the [`coordination-standard-0.13`](https://github.com/minqiyang/coordination-standard/tree/coordination-standard-0.13) branch and is no longer maintained.

`archive/` (historical drafts, no authority) and `project-notes/` (project handoffs) are local only and not in git.
