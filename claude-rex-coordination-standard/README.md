# Rex Coordination Standard

**Version `0.2.0`**

A skeleton for running several AI coding agents on one project. One coordinator agent runs the work, producers write code in Tabs of the [Rex](https://rex.superlogical.com) terminal, and you choose the agent CLI and the model of every role. During development, a producer edits and tests as freely as it needs. Before code is first put to real use, QA checks one frozen commit, and the coordinator decides whether a fresh read-only reviewer checks it too. The coordinator asks you only when the work is done, when it is stuck, or when only you can answer.

| File | What it holds | Share it? |
|---|---|---|
| [`coordinator.md`](coordinator.md) | Every rule: roles, the owner, cards, the gate, publication, the record, and the Rex commands | Yes, portable |
| [`bin/rex_agent.sh`](bin/rex_agent.sh) | The script that starts a producer in a Rex Tab, sends it a card, confirms the delivery, and waits for the status that the producer reports | Yes, portable |
| [`model_bindings.json`](model_bindings.json) | The model and the effort range of the producer and the reviewer, the reviewer's fallback, and the flags of each agent CLI | Personal; write your own |

This README only explains. If it disagrees with `coordinator.md`, `coordinator.md` wins. The card is a skeleton. Project rules govern, and a project adds its own stricter rules, for example for runs whose numbers it reports.

## Agents and models

![Agents and models](assets/agents_and_models.svg)

- **Coordinator.** Any agent CLI that runs shell commands and can run a command as a background task that wakes it when the command exits. You start it yourself, in a Rex Tab or in another terminal or app, with the model you choose. Today the standard runs with Claude Code as the coordinator.
- **Producer.** Writes the code, in its own git worktree, in its own Rex Tab. The coordinator chooses its effort within the range in `model_bindings.json`.
- **Reviewer.** Reads one frozen commit and reports findings. It runs headless and read-only, with no Tab. The coordinator chooses its effort within the range in `model_bindings.json`, by what an error in the change would cost.
- **Fallback.** If the reviewer's model cannot run, for example when its quota is used up, the coordinator uses the fallback model and continues.

## Who does what

![System architecture](assets/system_architecture.svg)

The coordinator is the agent session that you talk to. It writes cards, starts producers, checks their results, runs the gate, and keeps `tasks.md` current. It merges only within your authorization or the project's. It does not write project code, unless you tell it to for one specific change. While workers run, it waits on background tasks and does not poll.

## How a task moves

![Task flow](assets/workflow_lifecycle.svg)

1. **Card.** The card states the goal, the acceptance criteria, and the constraints. It does not state the method, unless you specified one.
2. **Dev loop.** The producer edits, runs, and reads results in its own worktree as often as it needs. The loop has no gate. The coordinator steers only when the producer asks or blocks, or when you change the requirements.
3. **Freeze.** When the work is first put to real use, the coordinator freezes it as a commit SHA. Real use is a merge or PR, a deploy or publication, a run whose numbers the project reports, an irreversible action, or input that other work builds on.
4. **QA.** The project's tests and checks run on that SHA in a clean detached worktree.
5. **Review.** The coordinator decides the review by what an error would cost. A small change that is easy to undo gets the coordinator's own check of the diff and the QA output. A change whose error could spoil results, corrupt state, or be hard to undo gets a reviewer: a new read-only session that knows the SHA, the criteria, and nothing about how the author worked.
6. **Findings.** A finding blocks only when it is MATERIAL: it shows a concrete scenario in this project by which the failure would occur in practice, and a significant effect on a result or a decision. Other findings are recorded.
7. **Accept and use.** When QA passes and no MATERIAL finding is open, the coordinator accepts that exact SHA and writes it in `tasks.md`. A repair makes a new SHA, which gets QA again.

## What the owner sees

Running work belongs to the coordinator. It decides order, timing, worktrees, retries, and whether and how deeply to review. It does not report progress. It stops and asks you only in three cases: all the work is done, it and its workers cannot proceed, or only you can answer the question, such as what a requirement means or a choice that changes what the project measures or ships.

To look at the work at any time, read `tasks.md` in the coordination folder, or open the Rex session `W-<project>` in the Rex app.

## The record and authority

The coordination folder is outside the repository, by default the sibling `<repo>.coord/`. It holds `tasks.md`, the cards and reports of each task, and the worktrees under `wt/`. `tasks.md` is the only record that the card requires. It is a live board with your standing authorization and one entry per open task.

Acceptance is not authorization. Push, merge, deploy, and irreversible actions need your authorization or the project's. Only the coordinator publishes, and only the accepted SHA, with a squash merge bound to that SHA.

## What you need

- The Rex app, which runs the Rex server, and the `rex` command on `PATH`.
- `jq` and `bash`.
- An agent CLI for each harness in `model_bindings.json`. A CLI that runs in a Tab must report its status to the terminal.

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

## Getting started

Open the Rex app, so that the Rex server runs. Then start your coordinator agent CLI and send it this prompt. Replace `<dir>` with the path of this folder, and do not copy the files.

```text
Read <dir>/coordinator.md, <dir>/model_bindings.json, and <dir>/bin/rex_agent.sh
as the coordinator policy, within the owner's authorization and project rules.
Declared version: 0.2.0.
Do not load coordination-standard/ or archive/. Follow the card.
```

## Setting up agents and models

The coordinator is the CLI you start, with the model you start it with. It is not in `model_bindings.json`. For the producer and the reviewer, edit `model_bindings.json` and set `updated`. No rule in `coordinator.md` changes.

- To move a role to another model, edit `roles.<role>.model`. The model must be in `models`.
- To change the effort that the coordinator may choose, edit `effort_min` and `effort_max` of the role.
- To change the reviewer's fallback, edit `roles.reviewer.fallback`.
- To add a model, add it to `models` with the harness that runs it. To use a newer version of a model, edit `models.<name>.id`.
- To add an agent CLI, add a harness to `harnesses`. Its key is the command name of the CLI.

A harness lists the flags that the card puts on the command line. `model_arg` sets the model. `effort_arg` sets the effort, and a harness whose model id carries the effort has none. `permission_args` are the permissions of a producer. `review_args` start a reviewer read-only, and `review_resume_args` resume one.

A producer needs a CLI that can commit in a linked git worktree. Codex under `-s workspace-write` cannot, because the worktree's git metadata lives in the main repository.

After a CLI update, check its flags with that CLI's `--help`.

## The Herdr card and coord-guard

The Herdr Coordinator Card 0.12.4 in [`../claude-herdr-coordination-standard/`](../claude-herdr-coordination-standard/) is a different, longer policy, with lanes, review seats, and more records. This card does not include those rules.

[`../guard/`](../guard/) holds `coord-guard`, a hook that checks three rules of the Herdr card. Two of them read files that this card does not require: QA logs with fixed names, and `ACCEPT` lines in `decisions.md`. Do not register the hook for a coordinator that follows this card.

## What was tested

Tested on 2026-11-17 with Rex `0.1.0-preview.14`:

- Claude Code 2.1.326: every command of `rex_agent.sh`, and its exit codes 0, 2, 3, and 4.
- Antigravity CLI 1.5.10 in a Tab: status reports, the folder-trust dialog, and one task.
- Codex 0.166.0 in a Tab: status reports (`idle`, `working`, `error`). The first Enter did not submit the text, and the second Enter did.

Tested on 2026-11-17 for the reviewer fallback, with Antigravity CLI 1.5.10 and the model `gemini-4-carbon-medium`:

- `agy -p` with `--mode plan` returned a reply and exit status 0.
- In `--mode plan`, the agent did not create a file that the prompt asked for.
- In `--mode plan`, in a git repository, the agent ran `git rev-parse` and `git diff --name-only` and returned their output.

Open points:

- No full task ran under this card: no gate, no review, and no two producers at the same time.
- No review ran with the reviewer command of either harness under this card.
- A Codex task from `working` to `done` did not run in a Tab.
- The exit codes 5, 6, and 7 of `rex_agent.sh` did not occur in a test.
- The permission dialogs of Claude Code did not run. The test covered a Claude Code question and the folder-trust dialog of Antigravity CLI.
