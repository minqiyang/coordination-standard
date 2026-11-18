# Coordination Standard

The live policy is the Rex Coordinator Card 0.2.0 in [`claude-rex-coordination-standard/`](claude-rex-coordination-standard/). The Herdr Coordinator Card 0.12.4 is a longer policy with lanes and review seats, and is kept in [`claude-herdr-coordination-standard/`](claude-herdr-coordination-standard/).

A skeleton for running several AI coding agents on one project. One coordinator agent runs the work. During development, producers edit and test as freely as they need. Before code is first put to real use, QA checks one frozen commit, and the coordinator decides whether a fresh read-only reviewer checks it too. Results come only from code that passed that check.

Every role is a slot that you fill with the agent CLI and the model you choose. The coordinator can be any agent CLI that can wait on background tasks, such as Claude Code. A producer runs on an agent CLI that the coordinator starts in a Rex Tab. A reviewer runs headless and read-only. The rules name roles, never models.

![Agents and models](claude-rex-coordination-standard/assets/agents_and_models.svg)

Producers run in the Rex session `W-<project>`. The coordinator interrupts you only when all the work is done, when it and its workers are stuck, or when only you can answer.

![System architecture](claude-rex-coordination-standard/assets/system_architecture.svg)

![Task flow](claude-rex-coordination-standard/assets/workflow_lifecycle.svg)

- Policy files: [`coordinator.md`](claude-rex-coordination-standard/coordinator.md), [`model_bindings.json`](claude-rex-coordination-standard/model_bindings.json), [`bin/rex_agent.sh`](claude-rex-coordination-standard/bin/rex_agent.sh)
- Start: [`claude-rex-coordination-standard/README.md`](claude-rex-coordination-standard/README.md)
- Guard for the Herdr card: [`guard/`](guard/), a hook for a Claude Code coordinator that blocks three repeated mistakes (seat without read-only sandbox, merge of an unaccepted SHA, seat before QA passes). Do not register it for a coordinator that follows the Rex card

The earlier Coordination Standard 0.13.0 is kept on the [`coordination-standard-0.13`](https://github.com/minqiyang/coordination-standard/tree/coordination-standard-0.13) branch and is no longer maintained.

`archive/` (historical drafts, no authority) and `project-notes/` (project handoffs) are local only and not in git.
