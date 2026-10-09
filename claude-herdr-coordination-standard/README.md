# Claude-herdr Coordination Standard

**Version `0.10.0`** (derived from Coordination Standard 0.9.0)

A working standard for running several AI coding agents on one project when the coordinator is a Claude Code session in a Herdr Tab. The coordinator plans, dispatches cards, verifies, gates, records, and merges when authorized. It writes no candidate code unless the owner names that change. For any other coordinator, use [Coordination Standard 0.13.0](../coordination-standard/README.md).

- One worker owns a whole dev loop. It edits, runs, and reads results in its own git worktree as often as it needs. It stops to report at the checkpoints its card names, or when it is blocked. At a checkpoint the coordinator replies with evidence, findings, priorities, or owner changes, never code.
- The lane's gate runs once, at the use point. A use point is the first time a frozen commit becomes load-bearing, such as a run of record, a merge or PR, a publication, or input that another card builds on. Dev rounds get no gate and never count as failures.
- Runs of record use only accepted code. Numbers from dev runs carry the label "unaccepted code" and never feed a result.
- GPT seats run headless and read-only, as background `codex exec` tasks. Producers, integrators, Claude seats, and adjudicators each get a new Tab and git worktree in a separate `workers` Herdr session.

## Why this standard

As the owner, you watch one thing: the coordinator in your main Herdr sidebar. The coordinator sends out producers, review seats, adjudicators, and integrators. These workers start, report, and close often, and they must not pull your attention away from more important work.

So the workers run where your sidebar does not show them. Most run in a separate Herdr session named `workers`, and GPT seats run headless. The coordinator tracks every worker. It interrupts you only when all the work is done, or when it and its workers cannot go on because of missing authority, a choice of meaning only you can make, the failure limit, or a blocker.

![Owner focus](assets/owner_focus.svg)

| File | What it holds | Share it? |
|---|---|---|
| [`coordinator.md`](coordinator.md) | Every rule: roles and Tabs, lanes and use points, routes and launching, cards and dev loops, QA and review, the failure limit, runs of record, plans, merge and publication. Names routes, never models | Yes, portable |
| [`model_bindings.json`](model_bindings.json) | Each route's default model and effort, and its replacement if it has one. The harness flags, also for headless seats. What the coordinator may adjust | Personal; write your own |

This README only explains. If it disagrees with the card, the card wins.

## How it fits together

![System architecture](assets/system_architecture.svg)

The coordinator launches every worker from `model_bindings.json`. Producers, integrators, Claude seats, and adjudicators each run in a new Tab in the `workers` Herdr session, on their own git worktree, and keep running if the coordinator ends. To look at them, run `herdr session attach workers` in another terminal window. A GPT seat runs as a background `codex exec -s read-only` task in its own detached worktree, with no Tab. Every seat reviews a detached worktree at the frozen commit.

The coordinator waits in the background. The exit of a `herdr agent prompt --wait` task, or of a headless seat's `codex exec`, wakes it. `tasks.md`, `decisions.md`, each launch's `launch.md`, cards, reports, QA logs, `runs/`, and the worktrees under `wt/` live in `<coord>`, outside the repository.

## Lifecycle

![Workflow lifecycle](assets/workflow_lifecycle.svg)

A card goes to one worker, which runs its dev loop and stops to report at the card's checkpoints or when it is blocked. At the first use point, the lane's gate runs once ([card §2](coordinator.md#2-lanes-and-gates)):

1. Freeze the candidate as a commit SHA.
2. Run deterministic QA at that SHA.
3. Run the lane's seats. ROUTINE gets a coordinator check and no seat, ELEVATED gets a REVIEW seat, and CRITICAL gets AUDIT and AUDIT_2.
4. Accept that exact SHA, then use it within recorded authorization.

Before the use point, the coordinator may send a frozen commit to one interim seat. Its findings are evidence and go through triage, and an unresolved MATERIAL finding blocks acceptance. An interim review never fails an attempt and never counts as the gate.

An attempt fails when the candidate frozen for the use point fails QA, or when the coordinator's check or a review round reports a MATERIAL finding that survives triage and any dispute. It also fails when the author stops without a candidate because it could not meet a criterion. The same worker repairs the candidate. The repair is a new candidate that needs new QA, and a fresh review if the lane has seats. After 2 failed attempts the card goes to EXPERT. After 2 failed EXPERT attempts the coordinator asks the owner ([card §5](coordinator.md#5-qa-review-acceptance)).

A run of record uses only accepted code, from a clean detached worktree. Each input it loads from outside that worktree is a frozen copy with a sha256 manifest. Before the start and before every resume, the coordinator checks that HEAD is the accepted SHA, that the tree is clean, and that the manifest verifies. Outputs go outside the worktree ([card §5](coordinator.md#5-qa-review-acceptance)).

## Getting started

Open a Herdr Tab, start Claude Code there, and send it this prompt. Replace `<dir>` with the path of this folder, and do not copy the files.

```text
Read <dir>/coordinator.md and <dir>/model_bindings.json as the coordinator policy,
within the owner's authorization and project rules. Declared version: 0.10.0.
Do not load coordination-standard/ or archive/. Follow the card.
```

## Switching a running coordinator

To move a coordinator that already runs an earlier version, send it this prompt:

```text
Switch to Claude-herdr 0.10.0. Re-read <dir>/coordinator.md and
<dir>/model_bindings.json as the coordinator policy, within the owner's
authorization and project rules. Do not load coordination-standard/ or archive/.
Keep what is already valid: frozen SHAs, accepted candidates, finished seat
reports, and failed attempts already counted (a card already on EXPERT stays
there).
Let live workers and seats finish their current card; apply 0.10.0 from each
task's next step, and launch new GPT seats headless.
Before your next dispatch, update tasks.md with each open task's next step
under 0.10.0. Ask me only if a task needs my decision.
```

## Swapping a model

Edit `models.<name>.id`, or a route's `model`, `effort`, or `replacement`, in `model_bindings.json`, and set `updated`. Each edit also gets a new version number ([Versioning](../README.md#versioning)). No rule in the card changes. Check flags with `claude --help` and `codex --help` after CLI updates. Each harness has `model_arg`, `effort_arg`, and `permission_args`. The codex harness also has `seat_args` and `seat_resume_args`. They start and resume a headless seat read-only, and `-o {report}` writes its report or reply. Under `-s workspace-write`, a Codex session cannot commit in a linked worktree, because the worktree's git metadata lives in the main repository. So keep producer routes on the claude harness.

Route bindings are defaults. When the coordinator is highly confident that another model or effort fits a card's task better, it may adjust that one launch. The route's default model and the new model must both use a harness listed in `adjustments`, and only claude is listed. The new model must be in `models`, and the effort must be from `min_effort` to `max_effort`. It adjusts for task fit only, never to make a gate easier to pass, and logs the change and its reason in that session's `launch.md` ([card §3](coordinator.md#3-routes-and-launching)). Routes on other harnesses, such as the GPT seats, stay as bound. If a route's model cannot run, the coordinator uses the route's `replacement` once, if it has one. Otherwise it pauses the task and tells the owner. To change what the coordinator may adjust, edit `adjustments`.
