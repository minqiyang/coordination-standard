# Coordination Standard

The live policy in this repository is the Rex Coordinator Card 0.3.0 in [`claude-rex-coordination-standard/`](claude-rex-coordination-standard/). A separate, longer policy with lanes and review seats, the Herdr Coordinator Card 0.12.4, lives in [`claude-herdr-coordination-standard/`](claude-herdr-coordination-standard/).

This repository defines a skeleton for running several AI coding agents on one project under a single coordinator. While code is still under development, producers edit and test as freely as they need without a gate on every change. Once code is first put to real use, QA checks one frozen commit and the coordinator decides whether a fresh read-only reviewer needs to inspect it too, so reported results come only from code that passed that check.

Rather than binding the rules to specific tools, each role is a slot that you fill with the agent CLI and the model you choose. Any agent CLI that can wait on background tasks, such as Claude Code, can serve as the coordinator. A producer runs on an agent CLI that the coordinator starts in a Rex Tab, while a reviewer runs headless and read-only.

![Agents and models](claude-rex-coordination-standard/assets/agents_and_models.svg)

Producers run in the Rex session `W-<project>`, and the coordinator interrupts you only when all the work is done, when it and its workers are stuck, or when only you can answer.

![System architecture](claude-rex-coordination-standard/assets/system_architecture.svg)

![Task flow](claude-rex-coordination-standard/assets/workflow_lifecycle.svg)

- Policy files: [`coordinator.md`](claude-rex-coordination-standard/coordinator.md), [`model_bindings.json`](claude-rex-coordination-standard/model_bindings.json), [`bin/rex_agent.sh`](claude-rex-coordination-standard/bin/rex_agent.sh)
- Start: [`claude-rex-coordination-standard/README.md`](claude-rex-coordination-standard/README.md)
- Guard for the Herdr card: [`guard/`](guard/), a hook for a Claude Code coordinator that blocks three repeated mistakes (a seat without a read-only sandbox, merge of an unaccepted SHA, or a seat before QA passes). Do not register it for a coordinator that follows the Rex card

The earlier Coordination Standard 0.13.0 remains on the [`coordination-standard-0.13`](https://github.com/minqiyang/coordination-standard/tree/coordination-standard-0.13) branch and is no longer maintained.

Two local-only directories are excluded from git: `archive/` holds historical drafts with no authority, and `project-notes/` holds project handoffs.
