# Rex Coordination Standard

**Version `0.3.0`**

This is a skeleton for running several AI coding agents on one project under a single coordinator. Producers write code in Tabs of the [Rex](https://rex.superlogical.com) terminal, and you choose the agent CLI and model for every role. During development, there is no gate, so a producer edits and tests in its own worktree as freely as it needs. Checks begin only when code is first put to real use, at which point QA tests one frozen commit and the coordinator decides whether a fresh read-only reviewer should check it too. The coordinator asks you only when the work is done, when it is stuck, or when only you can answer.

| File | What it holds | Share it? |
|---|---|---|
| [`coordinator.md`](coordinator.md) | All the rules: roles, the owner, cards, the gate, publication, the record, and the Rex commands | Yes, portable |
| [`bin/rex_agent.sh`](bin/rex_agent.sh) | The script that starts a producer in a Rex Tab, sends it a card, confirms delivery, and waits for its reported status | Yes, portable |
| [`model_bindings.json`](model_bindings.json) | The model and effort range for the producer and the reviewer, the reviewer's fallback, and the flags and update command for each agent CLI | Personal, write your own |

This README only explains how the standard works. If it disagrees with `coordinator.md`, `coordinator.md` wins. Because the card is a skeleton, owner authorization and project rules govern, and a project can add its own stricter rules, such as extra requirements for runs whose numbers it reports.

## Agents and models

![Agents and models](assets/agents_and_models.svg)

- **Coordinator.** Any agent CLI that can run shell commands and launch a background task that wakes it when the command exits. You start it yourself in a Rex Tab or in another terminal or app, using the model you choose. Today the standard runs with Claude Code as the coordinator.
- **Producer.** Writes the code in its own git worktree, inside its own Rex Tab. The coordinator picks its effort within the range in `model_bindings.json`.
- **Reviewer.** Reads one frozen commit and reports findings. It runs headless and read-only with no Tab, and the coordinator picks its effort within the range in `model_bindings.json` based on what an error in the change would cost.
- **Fallback.** If the reviewer's primary model cannot run because its quota is used up or it hits an outage, the coordinator switches to the fallback model and continues.

## Who does what

![System architecture](assets/system_architecture.svg)

The coordinator is the agent session you talk to directly. It writes cards, starts producers, verifies their results against files, tests, and git state, runs the gate, and keeps `tasks.md` current. It merges only within your authorization or the project's, and it does not write project code unless you tell it to for one specific change. While workers are running, it waits on background tasks instead of polling.

## How a task moves

![Task flow](assets/workflow_lifecycle.svg)

1. **Card.** Each card gives the whole task in one dispatch by stating the goal, the acceptance criteria, and the constraints. It names the finish line rather than the method, unless you specified a method.
2. **Dev loop.** The producer edits, runs, and reads results in its own worktree as often as it needs. Because development has no gate, the coordinator steers only when the producer asks or blocks, or when you change the requirements.
3. **Freeze.** When work is first put to real use, the coordinator freezes it as a commit SHA so every check targets that exact state. Real use means a merge or PR, a deploy or publication, a run whose numbers the project reports, an irreversible action, or input that other work builds on.
4. **QA.** The project's tests and checks run on that SHA in a clean detached worktree, isolated from the producer's working files.
5. **Review.** The coordinator decides how to review based on what an error would cost. For a small change that is easy to undo, its own check of the diff and the QA output is enough. For a change whose error could spoil results, corrupt state, or be hard to undo, it starts a fresh read-only reviewer session that receives the SHA, the acceptance criteria, and the MATERIAL rule, without being told how the author built or tested the work.
6. **Findings.** A finding blocks only when it is MATERIAL, meaning it shows both a concrete scenario in this project by which the failure would occur in practice and a significant effect on a result or a decision. Other findings are recorded so work can move on. A report that is empty, or that holds only a plan or a request for approval, does not count as a review, even when the reviewer exits 0. The coordinator runs the reviewer once more, and if that report does not count either, it asks you and does not accept the SHA.
7. **Accept and use.** Once QA passes and no MATERIAL finding is open, the coordinator accepts that exact SHA and writes it in `tasks.md`. Any repair creates a new SHA, which goes through QA again and has its review decided the same way.

## What the owner sees

Running the work belongs to the coordinator. It decides order, timing, worktrees, retries, and whether and how deeply to review, and it does not send progress reports. It stops and asks you only in three cases: when all the work is done, when it and its workers cannot proceed, or when only you can answer the question, such as what a requirement means or a choice that changes what the project measures or ships.

To look at the work at any time, read `tasks.md` in the coordination folder, or open the Rex session `W-<project>` in the Rex app.

## The record and authority

The coordination folder lives outside the repository, by default in the sibling `<repo>.coord/`, and holds `tasks.md`, the cards and reports for each task, and the worktrees under `wt/`. `tasks.md` is the only record the card requires. It is a live board edited in place with your standing authorization and one entry per open task.

Acceptance is not authorization. Push, merge, deploy, and irreversible actions require your authorization or the project's. Only the coordinator publishes, and only the accepted SHA, using a squash merge bound to that exact SHA. If the PR head changes after acceptance, the new head is a new SHA, and it goes through QA, a review decision, and acceptance before the merge.

## What you need

- The Rex app, which runs the Rex server, and the `rex` command on `PATH`.
- `jq` and `bash`.
- An agent CLI for each harness in `model_bindings.json`. Any CLI that runs in a Tab must report its status to the terminal.

The coordinator never starts a Rex server itself, because a server started from an agent shell gets the macOS file permissions of that agent and loses them when the agent binary is replaced. The server started by the Rex app has its own permissions, which is why every `rex` command in the card carries `--autostart=false`.

## How `rex_agent.sh` reads agent status

An agent CLI reports its status to the Rex terminal as `idle`, `working`, `blocked`, `done`, or `error`. A `blocked` record also gives the reason (`permission`, `question`, or `auth`) and one line of text, and `rex_agent.sh` reads this record every 2 seconds.

| Command | What it does |
|---|---|
| `rex_agent.sh start <session> <agent> <cwd> -- <command>...` | Creates the session if it does not exist, starts the command in a new Tab, and waits until the agent reports a status |
| `rex_agent.sh prompt <session> <agent> <timeout seconds> <text>` | Sends one line of text, confirms the delivery, and waits until the turn ends |
| `rex_agent.sh wait <session> <agent> <timeout seconds> [state]...` | Waits until the agent reports one of the states |
| `rex_agent.sh get <session> <agent>` | Prints the status record |

Because some agents treat the first Enter as part of a pasted line rather than a submit, `rex_agent.sh prompt` confirms that the text was delivered. It sends the text, presses Enter, and waits 10 seconds for a new status record. If no record arrives, it presses Enter one more time, and if the agent still does not start, the script prints `agent_prompt_stalled` and exits 5 without ever sending the text twice.

The header of the script lists every exit code.

## Getting started

Open the Rex app first so that the Rex server is running. Update your coordinator's agent CLI to its latest version, because the card updates only the CLIs of the workers it starts. Then start your coordinator agent CLI and send it this prompt, replacing `<dir>` with the path of this folder rather than copying the files.

```text
Read <dir>/coordinator.md, <dir>/model_bindings.json, and <dir>/bin/rex_agent.sh
as the coordinator policy, within the owner's authorization and project rules.
Declared version: 0.3.0.
Do not load coordination-standard/ or archive/. Follow the card.
```

## Setting up agents and models

The coordinator is whichever CLI and model you start yourself, so it is not listed in `model_bindings.json`. For the producer and the reviewer, edit `model_bindings.json` and update `updated`, while leaving `coordinator.md` unchanged.

- To move a role to another model, edit `roles.<role>.model` to point to an entry in `models`.
- To change the effort range the coordinator can choose from, edit `effort_min` and `effort_max` for the role.
- To change the reviewer's fallback, edit `roles.reviewer.fallback`.
- To add a model, add it to `models` with the harness that runs it, or edit `models.<name>.id` to use a newer version of a model.
- To add an agent CLI, add a harness to `harnesses` keyed by the command name of the CLI, with an `update_cmd` that installs its latest version.
- To change how a CLI is updated, edit `harnesses.<harness>.update_cmd`.

Each harness defines the flags that the card puts on the command line. `model_arg` sets the model, and `effort_arg` sets the effort unless the harness's model id already carries the effort. `permission_args` set the permissions of a producer, `review_args` start a reviewer read-only, and `review_resume_args` resume one.

A producer needs a CLI that can commit inside a linked git worktree. Codex running under `-s workspace-write` cannot do this because a linked worktree's git metadata lives in the main repository.

Workers run on the newest release of their CLI. Before the coordinator starts a producer or a reviewer, it runs that harness's `update_cmd` and waits for exit 0. An update replaces the CLI's files, so the coordinator skips it while another process of the same CLI runs, including the coordinator's own session. The worker then starts on the installed version. If the update fails twice, the coordinator does not start the worker on the old version: a reviewer moves to its fallback, and a producer goes to you. When an update installs a new version, the coordinator checks that every flag in the binding still appears in the CLI's `--help` before it starts the worker.

The right `update_cmd` depends on how the CLI was installed. On a Homebrew install of Claude Code, `claude update` only prints the `brew` command and exits 0 without updating, so the binding calls `brew upgrade claude-code@latest` instead. By default, Homebrew refreshes its version data only once every 24 hours, so the binding also sets `HOMEBREW_AUTO_UPDATE_SECS=0` to make `brew` refresh before each upgrade.

## The Herdr card and coord-guard

The Herdr Coordinator Card 0.12.4 in [`../claude-herdr-coordination-standard/`](../claude-herdr-coordination-standard/) is a different, longer policy with lanes, review seats, and more records. None of those rules are part of this card.

[`../guard/`](../guard/) holds `coord-guard`, a hook that checks three rules of the Herdr card. Two of those rules read files that this card does not require: QA logs with fixed names, and `ACCEPT` lines in `decisions.md`. So do not register the hook for a coordinator that follows this card.

## What was tested

Tested on 2026-11-17 with Rex `0.1.0-preview.14`:

- Claude Code 2.1.326: every command of `rex_agent.sh`, and its exit codes 0, 2, 3, and 4.
- Antigravity CLI 1.5.10 in a Tab: status reports, the folder-trust dialog, and one task.
- Codex 0.166.0 in a Tab: status reports (`idle`, `working`, `error`), where the first Enter did not submit the text and the second Enter did.

Tested on 2026-11-17 for the reviewer fallback, with Antigravity CLI 1.5.10 and the model `gemini-4-carbon-medium`:

- `agy -p` with `--mode plan` returned a reply and exit status 0.
- In `--mode plan`, the agent did not create a file that the prompt asked for.
- In `--mode plan` in a git repository, the agent ran `git rev-parse` and `git diff --name-only` and returned their output.

Tested on 2026-11-18 for the current reviewer and fallback models:

- Codex 0.166.0, and again Codex 0.170.0 after `codex update`, with `gpt-6.2-sol` at `medium` effort, using the reviewer's `-s read-only -C <worktree> -o <report>` flags: returned a reply and exit status 0.
- Antigravity CLI 1.5.11 with `gemini-4-hydrogen-high` in `--mode plan`: returned a reply and exit status 0. The agent did not run a `wc` command and did not create a file that the prompt asked for. It wrote a plan and asked for approval instead.
- With `--mode auto`, the same prompt ran the command and created the file, so that run was not read-only. `agy` 1.5.11 printed `--mode auto is not available for this account; continuing in the default mode.` In the log of a later run with `--mode auto`, that check comes before sign-in, and `SetCycleMode called: auto` comes after it.

Tested on 2026-11-18 for the update commands:

- `brew upgrade claude-code@latest` installed Claude Code 2.1.327 and exited 0. Run again with nothing to upgrade, it also exited 0. With `HOMEBREW_AUTO_UPDATE_SECS=0`, it printed `Auto-updating Homebrew...` before the upgrade check and exited 0.
- `claude update` on the Homebrew install printed the `brew` command and exited 0 without updating.
- `codex update` installed Codex 0.170.0 and exited 0.
- `agy update` reported that 1.5.11 is the latest version and exited 0.
- After these updates, every flag in `model_bindings.json` still appeared in `claude --help`, `codex exec --help`, `codex exec resume --help`, and `agy --help`.

Open points:

- No full task ran under this card, so there was no gate, no review, and no two producers running at the same time.
- No review ran with the reviewer command of either harness under this card.
- A Codex task from `working` to `done` did not run in a Tab.
- The exit codes 5, 6, and 7 of `rex_agent.sh` did not occur in a test.
- No full start ran under the update rule. The coordinator's check for another running process of the same CLI is untested.
- A headless `agy` run in `--mode plan` once returned no output and exit status 0, because `agy` auto-denied a command that the agent wanted to run. A fallback review on `agy` can therefore come back empty with exit 0. The card does not count such a report as a review, but that rule has not run in a real review.
- The permission dialogs of Claude Code did not run. The test covered a Claude Code question and the folder-trust dialog of Antigravity CLI instead.
