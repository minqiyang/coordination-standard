# Claude-herdr Coordination Standard

**Version `0.11.0`** (derived from Coordination Standard 0.9.0)

A standard for running several AI coding agents on one project when the coordinator is a Claude Code session in a Herdr Tab. During development, workers edit and test as freely as they need. Before frozen code is first put to real use, QA checks it once, and so do fresh read-only reviewers when the lane requires them. Results come only from code that passed that check. The earlier Coordination Standard 0.13.0, for any coordinator, is kept on the [`coordination-standard-0.13`](https://github.com/minqiyang/coordination-standard/tree/coordination-standard-0.13) branch and is no longer maintained.

| File | What it holds | Share it? |
|---|---|---|
| [`coordinator.md`](coordinator.md) | Every rule: roles and Tabs, lanes and use points, routes and launching, cards and dev loops, QA and review, the failure limit, runs of record, plans, merge and publication. Names routes, never models | Yes, portable |
| [`model_bindings.json`](model_bindings.json) | Each route's default model and effort, and its replacement if it has one. The harness flags, also for headless seats. What the coordinator may adjust | Personal; write your own |

This README only explains. If it disagrees with `coordinator.md`, `coordinator.md` wins.

## Who does what

![System architecture](assets/system_architecture.svg)

**The coordinator runs the project.** It is the Claude Code session in your Herdr sidebar. It plans the work, sends cards to workers, checks their results, runs the gates, and keeps the records. It merges only when you authorize it. It does not write code itself, unless you tell it to for one specific change. While workers run, it waits in the background until a worker replies or a headless seat exits.

**One long-running worker writes the code.** This worker is the producer. Its default route is GENERAL_EXEC. After two failed attempts, or when you ask, the card moves to EXPERT. By default, both are Opus sessions. Their model and effort are set in `model_bindings.json`. To combine accepted work from several cards, the coordinator starts a fresh GENERAL_EXEC integrator.

**Review seats read frozen code and write reports.** By default, REVIEW and AUDIT are GPT seats. They run read-only as background `codex exec` tasks, with no Tab. If GPT cannot run, that seat uses its Opus replacement once. AUDIT_2 is an Opus seat in a Tab. The ADJUDICATOR is not a seat. It is a fresh, read-only session, Opus by default, in a Tab that rules on disputed findings.

**Each worker has its own git worktree.** Workers with a Tab run in a separate Herdr session named `workers`, so your sidebar does not show them. They keep running if the coordinator ends. To look at them, run `herdr session attach workers` in another terminal window.

**The coordinator's subagents only read.** They scout, read reports and diffs, and check claims. A subagent may fill a seat on the claude harness, but only for a delta re-review ([Failures and escalation](#failures-and-escalation)) or when you ask for it. A subagent never fills any other role, such as the ADJUDICATOR.

**Model bindings are defaults.** The exact model and effort of each route live in `model_bindings.json`. When the coordinator is highly confident that another model or effort fits a card better, it may change that one launch. It does this only for task fit, never to make a gate easier to pass. The change must stay on the Claude side and within the effort range in `adjustments`. The coordinator writes the change and the reason in that launch's `launch.md`. GPT seats stay as bound. If a route with no replacement cannot run, the coordinator pauses the task and tells you. You set the defaults yourself ([Choosing models](#choosing-models)).

## How a task moves

![Task flow](assets/workflow_lifecycle.svg)

1. **Card.** The card states the goal, the finish line, and the acceptance criteria. It does not state the method. It also names the lane, the use points (step 3), the checkpoints, and the dev paths.
2. **Dev loop.** The worker edits, runs, and reads results in its own worktree as often as it needs. The loop has no gate, and nothing in it counts as a failure. The worker stops to report only at a checkpoint or when it is blocked. The coordinator replies with evidence, findings, priorities, or changes from you. It never sends code, or a method that you did not specify. If you change the requirements, the coordinator tells the worker at once. Every number from a dev run carries the label "unaccepted code".
3. **Use point.** A use point is the first time the worker's code is put to real use. Examples are a run of record, a merge or PR, a publication, an irreversible action, and input that another card builds on. Here the coordinator freezes the code as a commit SHA, but only after the worker has made every change this use needs. For a run of record, this includes code that writes its outputs outside the worktree and loads outside inputs from frozen copies. The coordinator does not gate code that must still change. For an earlier look, it can send the code to an interim seat (step 5). This frozen code is the candidate. The gate runs once for that SHA. QA runs first. Then the coordinator starts all of the lane's seats at the same time. If the gate passes, the coordinator accepts that exact SHA and uses it within your authorization. If the gate fails, the worker repairs the candidate ([Failures and escalation](#failures-and-escalation)). If a small change is still needed after the gate, a delta re-review checks it.
4. **Lane.** The lane sets how much review the gate needs:

   | Lane | When | Gate |
   |---|---|---|
   | ROUTINE | Ordinary, local work that is easy to undo | QA and a coordinator check |
   | ELEVATED | An important change to behavior or correctness | QA and a REVIEW seat |
   | CRITICAL | An error could spoil results, corrupt state, or cause effects that cannot be undone | QA, an AUDIT seat, and an AUDIT_2 seat |

   A lane can rise but never drop. Work that merges by PR is at least ELEVATED. Structural work is CRITICAL. [`coordinator.md` §2](coordinator.md#2-lanes-and-gates) gives the full rules.
5. **Interim seat.** Before the use point, the coordinator may freeze an interim candidate and send it to one seat. It should do this early when the dev loop steers by numbers from the card's own code, such as a scorer. The seat's findings go through triage like any others, and an open MATERIAL finding blocks acceptance. An interim review never fails an attempt and does not count as the gate.

## How review stays independent

- Each seat is a new session and is read-only. It does not see the worker's own notes or context. In the first round it does not see other seats' reports.
- The seat reviews a detached worktree at the candidate's SHA. Every seat card uses one template. Its scope is the full diff plus the frozen copies of outside inputs. The card never tells how the worker built or tested the candidate. It never narrows the scope, except in a delta re-review ([Failures and escalation](#failures-and-escalation)).
- After the report arrives, the coordinator checks that the worktree is still clean at the SHA and that the seat's launch matches its binding. If not, the seat does not count.
- A finding is MATERIAL only when it shows both a concrete trigger scenario in this project and a measurable, significant impact on a main decision or result. Any other finding is ADVISORY. ADVISORY findings are recorded but never block.
- The coordinator triages each MATERIAL finding first. It may downgrade one that fails the MATERIAL rule, asks for work beyond the criteria, or is only a style preference. It may never downgrade by itself a finding that has a reproduction, a failing test, or other machine evidence.
- For any other disputed finding, the coordinator chooses how to settle it. It can ask the seat to withdraw the finding or show a trigger path, rule itself, or call a fresh ADJUDICATOR. In a CRITICAL gate, only the ADJUDICATOR may rule against a seat. Each downgrade and each ruling against a seat gets an `OVERRIDE` line in `decisions.md`.

## Failures and escalation

An attempt is one candidate frozen for a use point, or one repair of it. An attempt fails only when:

- the frozen candidate fails QA,
- a MATERIAL finding from the coordinator's check or from a review round is still valid after triage and any dispute, or
- the worker stops without a candidate because it could not meet a criterion.

Environment, permission, and quota problems do not count. Unclear requirements and measured results that are not criteria do not count either.

The same worker repairs the candidate, and it fixes only the open MATERIAL findings. Each repair is a new candidate, so it needs new QA and, if the lane has seats, a fresh review.

If the change since the last full gate is small, a delta re-review replaces the full gate. A full gate counts here only when every seat of the card's current lane reported on one SHA. QA still runs in full, but only one fresh seat reviews: AUDIT on CRITICAL, REVIEW on ELEVATED. AUDIT_2 does not review again, even when it raised the finding. The seat checks the whole change since the full gate, frozen inputs included, the code that change can affect, and all earlier MATERIAL findings.

A change is small only when it repairs MATERIAL findings, or fixes a blocking defect that nobody knew of at the freeze, and touches only the code those concern, plus tests. It must also add no feature and change no criterion, interface, schema, contract, or structural item. A change that the use needed and that the coordinator knew of at the freeze is never small. Neither is an integration, a merge of base commits, or a plan change that needs re-acceptance. The coordinator writes in `decisions.md` why the change is small, and the seat checks that reason. A bigger change gets the full gate again. ROUTINE work has no delta re-review.

After two failed attempts the card moves to EXPERT, which makes every later repair. After two failed EXPERT attempts, or when no new evidence or approach is left, the coordinator asks you.

## Runs of record

A run of record is a run whose output feeds a project result, another card or its gate, a report or benchmark, or one of your go/no-go decisions. To use a number from a dev run, run it again as a run of record.

- It runs only an accepted candidate, from a clean detached worktree at its SHA.
- Each input from outside that worktree, such as a skill, prompt, or config, is a frozen copy with a sha256 manifest.
- Before the start and before every resume, the coordinator checks that HEAD is the accepted SHA, that the tree is clean, and that the manifest verifies. Outputs go outside the worktree.
- The coordinator approves and starts runs of record itself. It asks you only if the run is also an irreversible action.

## Records and authority

The records live in `<coord>`, a folder outside the repository. Cards, reports, QA logs, and the worktrees under `wt/` live there too.

| Record | What it holds |
|---|---|
| `tasks.md` | The live board, edited in place, and your standing authorization |
| `decisions.md` | Each task's downgrades, rulings, risk acceptances, plan acceptances, reasons a change counts as small, and publication results |
| `launch.md` | The exact command of each launch, and any adjustment or replacement |
| `runs/` | Runs of record |

- Acceptance is not authorization. Push, merge, deploy, and irreversible actions need your authorization. Standing authorization goes in `tasks.md` once, and the coordinator does not ask for it again.
- Only the coordinator publishes, and only the accepted SHA. It merges with a squash merge bound to that SHA.

## What the owner sees

![Owner focus](assets/owner_focus.svg)

You watch only the coordinator. Workers start, report, and close often, and they run where your sidebar does not show them. The coordinator interrupts you only when all the work is done, or when it and its workers cannot go on. They cannot go on when authority is missing, when a choice of meaning needs you, when the failure limit is reached, or when another blocker stops them. The rest of the time it keeps working and keeps `tasks.md` current.

## Getting started

Open a Herdr Tab, start Claude Code there, and send it this prompt. Replace `<dir>` with the path of this folder, and do not copy the files.

```text
Read <dir>/coordinator.md and <dir>/model_bindings.json as the coordinator policy,
within the owner's authorization and project rules. Declared version: 0.11.0.
Do not load coordination-standard/ or archive/. Follow the card.
```

## Switching a running coordinator

To move a coordinator that already runs an earlier version, send it this prompt:

```text
Switch to Claude-herdr 0.11.0. Re-read <dir>/coordinator.md and
<dir>/model_bindings.json as the coordinator policy, within the owner's
authorization and project rules. Do not load coordination-standard/ or archive/.
Keep what is already valid: frozen SHAs, accepted candidates, finished seat
reports, and failed attempts already counted (a card already on EXPERT stays
there).
Let live workers and seats finish their current card; apply 0.11.0 from each
task's next step. Never gate a SHA that must still change before its use.
Start all of a lane's seats at the same time. After a full gate, give a small
change a delta re-review.
Before your next dispatch, update tasks.md with each open task's next step
under 0.11.0. Ask me only if a task needs my decision.
```

## Choosing models

You choose the model and effort of each route, and they do not have to be Claude models. The defaults are Opus for GENERAL_EXEC, EXPERT, AUDIT_2, and the ADJUDICATOR, and GPT-6.1 Sol at xhigh for REVIEW and AUDIT. To change them, edit `model_bindings.json` and set `updated`. No rule in `coordinator.md` changes.

- To use a newer version of a model, edit `models.<name>.id`.
- To move a route to another model or effort, edit the route's `model`, `effort`, or `replacement`.
- To add a model, add it to `models` with the harness that runs it: `claude` for the Claude Code CLI, or `codex` for the Codex CLI.

Seats and the ADJUDICATOR can use either harness. A seat on the codex harness runs headless, and a seat on the claude harness runs in a Tab. Producer routes (GENERAL_EXEC and EXPERT) stay on the claude harness. Under `-s workspace-write`, a Codex session cannot commit in a linked worktree, because the worktree's git metadata lives in the main repository. The coordinator itself is always a Claude Code session.

The coordinator may also change one launch by itself, within `adjustments` ([Who does what](#who-does-what)). To change what it may adjust, edit `adjustments`.

After CLI updates, check the flags with `claude --help` and `codex --help`. Each harness has `model_arg`, `effort_arg`, and `permission_args`. The codex harness also has `seat_args` and `seat_resume_args`. They start and resume a headless seat read-only, and `-o {report}` writes its report or reply.
