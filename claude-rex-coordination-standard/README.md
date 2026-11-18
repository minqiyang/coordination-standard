# Rex Coordination Standard

**Version `0.1.0`** (derived from Herdr Coordination Standard 0.12.4)

A standard for running several AI coding agents on one project. One coordinator agent runs the work, each worker runs in a Tab of the [Rex](https://rex.superlogical.com) terminal, and you choose the agent CLI and the model of every role. During development, workers edit and test as freely as they need. Before frozen code is first put to real use, QA checks it once, and so do fresh read-only reviewers when the lane requires them. Results come only from code that passed that check. The earlier Coordination Standard 0.13.0 is kept on the [`coordination-standard-0.13`](https://github.com/minqiyang/coordination-standard/tree/coordination-standard-0.13) branch and is no longer maintained.

| File | What it holds | Share it? |
|---|---|---|
| [`coordinator.md`](coordinator.md) | Every rule: roles and Tabs, lanes and use points, routes and launching, cards and dev loops, QA and review, the failure limit, runs of record, plans, merge and publication. Names routes, never models | Yes, portable |
| [`bin/rex_agent.sh`](bin/rex_agent.sh) | The script that starts a worker in a Rex Tab, sends it a card, confirms the delivery, and waits for the status that the worker reports | Yes, portable |
| [`model_bindings.json`](model_bindings.json) | Each route's default model and effort, and its replacement if it has one. Each harness (the agent CLI that runs a model) and its flags, also for headless seats. What the coordinator may adjust | Personal; write your own |

This README only explains. If it disagrees with `coordinator.md`, `coordinator.md` wins.

## Agents and models

![Agents and models](assets/agents_and_models.svg)

Every role is a slot, and you choose the agent CLI and the model that fill it. `coordinator.md` names roles and never a model. `model_bindings.json` gives each role a model and an effort.

- **Coordinator.** Any agent CLI that runs shell commands and can run a command as a background task that wakes it when the command exits. You start it yourself, in a Rex Tab or in another terminal or app, with the model you choose. Today the standard runs with Claude Code as the coordinator.
- **Workers.** Each model in `model_bindings.json` names its harness, the agent CLI that runs it. A harness name is the command name of that CLI, such as `claude` or `codex`. The coordinator starts each worker in a Rex Tab with `bin/rex_agent.sh`. The CLI must report its status to the terminal ([How `rex_agent.sh` reads agent status](#how-rex_agentsh-reads-agent-status)).
- **Two models.** The diagrams use two model slots. Model 1 is the main model. It writes the code (GENERAL_EXEC and EXPERT) and fills AUDIT_2 and the ADJUDICATOR. Model 2 fills REVIEW and AUDIT, so the first review of each gate comes from a model that did not write the code. You can also use one model for every role, or more than two.

[Setting up agents and models](#setting-up-agents-and-models) tells how to change them.

## Who does what

![System architecture](assets/system_architecture.svg)

**The coordinator runs the project.** It is the agent session that you talk to. It plans the work, sends cards to workers, checks their results, runs the gates, and keeps the records. It merges only when you authorize it. It does not write code itself, unless you tell it to for one specific change. While workers run, it waits on background tasks until a worker replies or a headless seat exits.

**One long-running worker writes the code.** This worker is the producer. Its default route is GENERAL_EXEC. After two failed attempts, or when you ask, the card moves to EXPERT. To combine accepted work from several cards, the coordinator starts a fresh GENERAL_EXEC integrator.

**Review seats read frozen code and write reports.** REVIEW is the seat of an ELEVATED gate. AUDIT and AUDIT_2 are the two seats of a CRITICAL gate. A seat on the codex harness runs read-only as a background `codex exec` task, with no Tab. Every other seat runs in a Tab. If a seat's model cannot run, the seat uses its replacement once. The ADJUDICATOR is not a seat. It is a fresh, read-only session in a Tab that rules on disputed findings.

**Each worker has its own git worktree.** Workers with a Tab run in a Rex session named `W-<project>`. They keep running if the coordinator or the Rex app ends. To look at them, open that session in the Rex app.

**The coordinator's subagents only read.** If the coordinator's agent CLI has subagents, they scout, read reports and diffs, and check claims. A subagent may fill a seat that is bound to the coordinator's own CLI, but only for a delta re-review ([Failures and escalation](#failures-and-escalation)) or when you ask for it. A subagent never fills any other role, such as the ADJUDICATOR.

**Model bindings are defaults.** The exact model and effort of each route live in `model_bindings.json`. When the coordinator is highly confident that another model or effort fits a card better, it may change that one launch. It does this only for task fit, never to make a gate easier to pass. The change must stay within `adjustments`, which lists the harnesses it may change and the effort range. The coordinator writes the change and the reason in that launch's `launch.md`. If a route with no replacement cannot run, the coordinator pauses the task and tells you.

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

   Code that computes scores, judgments, costs, or billing that feed results is CRITICAL. A lane can rise but never drop. When ELEVATED rises to CRITICAL, the REVIEW report counts as AUDIT and only AUDIT_2 is added. Work that merges by PR is at least ELEVATED. Structural work is CRITICAL. [`coordinator.md` §2](coordinator.md#2-lanes-and-gates) gives the full rules.
5. **Interim seat.** Before the use point, the coordinator may freeze an interim candidate and send it to one seat. It should do this early when the dev loop steers by numbers from the card's own code, such as a scorer. The seat's findings go through triage like any others, and an open MATERIAL finding blocks acceptance. An interim review never fails an attempt and does not count as the gate.

## How review stays independent

- Each seat is a new session and is read-only. It does not see the worker's own notes or context. In the first round it does not see other seats' reports.
- The seat reviews a detached worktree at the candidate's SHA. Every seat card uses one template. Its scope is the full diff plus the frozen copies of outside inputs. The card never tells how the worker built or tested the candidate. It never narrows the scope, except in a delta re-review ([Failures and escalation](#failures-and-escalation)).
- After the report arrives, the coordinator checks that the worktree is still clean at the SHA and that the seat's launch matches its binding. If not, the seat does not count.
- A finding is MATERIAL only when it shows both a concrete trigger scenario in this project and a measurable, significant impact on a main decision or result. Any other finding is ADVISORY. ADVISORY findings are recorded but never block.
- For a guard against the agent under test, such as a sandbox or answer-leak protection, seats assume a normal agent doing its task. A hole that only a command crafted on purpose can reach is ADVISORY: recorded, not blocking.
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
- When runs of record share a quota with seats, dev runs, or judges, the runs of record come first.

## Records and authority

The records live in `<coord>`, a folder outside the repository. Cards, reports, QA logs, and the worktrees under `wt/` live there too.

| Record | What it holds |
|---|---|
| `tasks.md` | The live board, edited in place, and your standing authorization |
| `decisions.md` | Each task's acceptances (`ACCEPT <SHA>` lines), downgrades, rulings, risk acceptances, plan acceptances, reasons a change counts as small, and publication results |
| `launch.md` | The exact command of each launch, and any adjustment or replacement |
| `runs/` | Runs of record |

- Acceptance is not authorization. Push, merge, deploy, and irreversible actions need your authorization. Standing authorization goes in `tasks.md` once, and the coordinator does not ask for it again.
- Only the coordinator publishes, and only the accepted SHA. It merges with a squash merge bound to that SHA. If the repository requires an up-to-date branch, it may merge a clean merge of the base into that SHA after QA passes on it, with no new review.

## What the owner sees

The coordinator decides process questions itself: order, timing, seat scheduling, worktrees, retries, and QA layout. It brings you only what changes results or exposure: the content that the project measures or ships (such as skills and prompts), which models the project's own runs use, criteria and meaning, and external actions without standing authorization.

You watch only the coordinator. Workers start, report, and close often, and they run in their own Rex session. The coordinator interrupts you only when all the work is done, or when it and its workers cannot go on. They cannot go on when authority is missing, when a choice of meaning needs you, when the failure limit is reached, or when another blocker stops them. The rest of the time it keeps working and keeps `tasks.md` current.

## The guard (optional)

[`guard/`](../guard/) holds `coord-guard`, a small Go program for a Claude Code coordinator. Claude Code runs it as a PreToolUse hook before each Bash command. It stops three mistakes that agents made again and again although the card forbade them:

- a `codex exec` seat without the read-only sandbox, including on resume;
- a `gh pr merge` that is not a squash bound to an SHA with an `ACCEPT` line in `decisions.md`;
- a `codex exec` seat launch before every QA log of its SHA ends with `exit status: 0`, or from a dirty worktree.

The guard checks only coordination work: a session that has `COORD_DIR` set, or whose working folder finds a coord folder. In all other work it allows every command, so it does not get in the way of normal tasks. The one exception is a seat launch with `-C` under `<coord>/wt/`, which R3 always checks.

Judgment rules stay in `coordinator.md`. The card applies in full without the guard, also with another coordinator CLI. The guard is a guardrail, not a security boundary. See [`guard/README.md`](../guard/README.md) to build and register it.

## What you need

- The Rex app, which runs the Rex server, and the `rex` command on `PATH`.
- `jq` and `bash`.
- An agent CLI for each harness in `model_bindings.json`. Each CLI that runs in a Tab must report its status to the terminal.

The coordinator never starts a Rex server. A Rex server that an agent shell starts gets the macOS file permissions of that agent, and loses them when the agent binary is replaced. The server that the Rex app starts has its own permissions. For this reason every `rex` command in the card carries `--autostart=false`.

## How `rex_agent.sh` reads agent status

An agent CLI reports its status to the Rex terminal: `idle`, `working`, `blocked`, `done`, or `error`. A `blocked` record also gives the reason (`permission`, `question`, or `auth`) and one line of text. `rex_agent.sh` reads this record every 2 seconds.

| Command | What it does |
|---|---|
| `rex_agent.sh start <session> <agent> <cwd> -- <command>...` | Creates the session if it does not exist, starts the command in a new Tab, and waits until the agent reports a status |
| `rex_agent.sh prompt <session> <agent> <timeout seconds> <text>` | Sends one line of text, confirms the delivery, and waits until the turn ends |
| `rex_agent.sh wait <session> <agent> <timeout seconds> [state]...` | Waits until the agent reports one of the states |
| `rex_agent.sh get <session> <agent>` | Prints the status record |

`rex_agent.sh prompt` confirms the delivery of the text. It sends the text, presses Enter, and waits 10 seconds for a new status record. If no record comes, it presses Enter one more time. If the agent still does not start, the script prints `agent_prompt_stalled` and exits 5. The script never sends the text twice.

The header of the script lists every exit code.

## What is different from the Herdr card

The rules for roles, lanes, gates, routes, cards, QA, review, and merge are the same in both cards. Only the commands that start, prompt, and watch workers are different.

| Herdr card | Rex card |
|---|---|
| The coordinator runs in a Herdr Tab and checks `HERDR_ENV` | The coordinator runs anywhere and checks `rex --autostart=false whoami` |
| Workers run in the Herdr session `workers`, in the workspace `W-<project>` | Workers run in the Rex session `W-<project>` |
| The coordinator starts a missing `workers` server | The coordinator never starts a Rex server. It tells the owner to open the Rex app |
| `herdr agent start` | `rex_agent.sh start` |
| `herdr agent prompt --wait --timeout` | `rex_agent.sh prompt` |
| `herdr agent wait` | `rex_agent.sh wait` |
| `herdr agent get` | `rex_agent.sh get` |
| `herdr agent read` | `rex capture --trim` |
| `send-keys` | `rex send-key` |
| `herdr agent list` | `rex block ls` |
| Timeouts in milliseconds | Timeouts in seconds |
| The owner's sidebar hides the `workers` session | The Rex app shows every session, also `W-<project>` |

## Getting started

Open the Rex app, so that the Rex server runs. Then start your coordinator agent CLI and send it this prompt. Replace `<dir>` with the path of this folder, and do not copy the files.

```text
Read <dir>/coordinator.md and <dir>/model_bindings.json as the coordinator policy,
within the owner's authorization and project rules. Declared version: 0.1.0.
Do not load coordination-standard/ or archive/. Follow the card.
```

## Switching a coordinator from the Herdr card

To move a coordinator that runs the Herdr Coordination Standard 0.12.4, send it this prompt:

```text
Switch to Rex Coordination Standard 0.1.0. Re-read <dir>/coordinator.md and
<dir>/model_bindings.json as the coordinator policy, within the owner's
authorization and project rules. Do not load coordination-standard/ or archive/.
Keep what is already valid: frozen SHAs, accepted candidates, finished seat
reports, and failed attempts already counted.
Let live workers in Herdr finish their current card, and handle them with
Herdr commands until they close. Start every new worker in Rex with
<dir>/bin/rex_agent.sh. Never start a Rex server.
Before your next dispatch, update tasks.md with each open task's next step.
Ask me only if a task needs my decision.
```

## Setting up agents and models

The coordinator is the CLI you start, with the model you start it with; it is not in `model_bindings.json`. For every other role, edit `model_bindings.json` and set `updated`. No rule in `coordinator.md` changes.

- To use a newer version of a model, edit `models.<name>.id`.
- To move a route to another model or effort, edit the route's `model`, `effort`, or `replacement`.
- To add a model, add it to `models` with the harness that runs it.
- To add an agent CLI, add a harness to `harnesses`. Its key is the command name of the CLI. It lists `model_arg`, `effort_arg`, and `permission_args`, the flags that set the model, the effort, and the permissions.

Producer routes (GENERAL_EXEC and EXPERT) need a CLI that can commit in a linked git worktree. Codex under `-s workspace-write` cannot, because the worktree's git metadata lives in the main repository. A seat on the codex harness runs headless: `seat_args` and `seat_resume_args` start and resume it read-only, and `-o {report}` writes its report or reply. Seats and the ADJUDICATOR on any other harness run in a Tab.

The coordinator may also change one launch by itself, within `adjustments` ([Who does what](#who-does-what)). To change what it may adjust, edit `adjustments`.

After a CLI update, check its flags with that CLI's `--help`.

## What was tested

Tested on 2026-11-17 with Rex `0.1.0-preview.14`:

- Claude Code 2.1.326: every command of `rex_agent.sh`, and its exit codes 0, 2, 3, and 4.
- Antigravity CLI 1.5.10: status reports, the folder-trust dialog, and one task.
- Codex 0.166.0: status reports (`idle`, `working`, `error`). The first Enter did not submit the text, and the second Enter did.

Open points:

- No full task ran under this card: no gate, no seat, and no two workers at the same time.
- A Codex task from `working` to `done` did not run.
- The exit codes 5, 6, and 7 of `rex_agent.sh` did not occur in a test.
- The permission dialogs of Claude Code and Codex did not run. The test covered a Claude Code question and the folder-trust dialog of Antigravity CLI.
- The coord-guard hook did not run against a coordinator that uses this card. The hook finds coordination work by the `<coord>` folder and checks `codex exec` and `gh pr merge` commands. It reads no Herdr command, so its three rules also apply under this card.
