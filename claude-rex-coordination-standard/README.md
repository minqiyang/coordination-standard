# Rex Coordination Standard

**Version `0.1.0`** (derived from Herdr Coordination Standard 0.12.4)

This folder holds the Rex version of the coordinator card. The rules for roles, lanes, gates, routes, cards, QA, review, and merge are the same as in the Herdr card. Only the commands that start, prompt, and watch workers are different.

| File | What it holds |
|---|---|
| [`coordinator.md`](coordinator.md) | Every rule. If this README disagrees with `coordinator.md`, `coordinator.md` wins |
| [`model_bindings.json`](model_bindings.json) | Each route's default model and effort, and the flags of each harness |
| [`bin/rex_agent.sh`](bin/rex_agent.sh) | The script that starts a worker in a Rex Tab, sends it a card, and waits for its status |

The Herdr card stays in `claude-herdr-coordination-standard/`. Its README explains the rules in full.

## What is different from the Herdr card

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

A Rex server that an agent shell starts gets the macOS file permissions of that agent, and loses them when the agent binary is replaced. The server that the Rex app starts has its own permissions. For this reason every `rex` command in the card carries `--autostart=false`.

## How `rex_agent.sh` reads agent status

An agent CLI reports its status to the Rex terminal: `idle`, `working`, `blocked`, `done`, or `error`. A `blocked` record also gives the reason (`permission`, `question`, or `auth`) and one line of text. `rex_agent.sh` reads this record every 2 seconds.

`rex_agent.sh prompt` confirms the delivery of the text. It sends the text, presses Enter, and waits 10 seconds for a new status record. If no record comes, it presses Enter one more time. If the agent still does not start, the script prints `agent_prompt_stalled` and exits 5. The script never sends the text twice.

The header of the script lists every exit code.

## What was tested

Tested on 2026-11-17 with Rex `0.1.0-preview.14`:

- Claude Code 2.1.326: every command and every exit code of `rex_agent.sh` except 5, 6, and 7.
- Antigravity CLI 1.5.10: status reports, the folder-trust dialog, and one task.
- Codex 0.166.0: status reports (`idle`, `working`, `error`). The first Enter did not submit the text, and the second Enter did.

Open points:

- No full task ran under this card: no gate, no seat, and no two workers at the same time.
- A Codex task from `working` to `done` did not run.
- The permission dialogs of Claude Code and Codex did not run. The test covered a Claude Code question and the folder-trust dialog of Antigravity CLI.
- The coord-guard hook in `guard/` did not run against a coordinator that uses this card. The hook finds coordination work by the `<coord>` folder and checks `codex exec` and `gh pr merge` commands. It reads no Herdr command, so its three rules also apply under this card.
