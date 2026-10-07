# Claude-herdr Coordination Standard

**Version `0.8.0`** (derived from Coordination Standard 0.9.0)

A working standard for running several AI coding agents on one project when the coordinator is a **Claude Code** session in a **Herdr** Tab. Producers, review seats, and integrators each run in their own Herdr Tab and git worktree; the coordinator dispatches, waits in the background, verifies, and gates. For any other coordinator, use [Coordination Standard 0.13.0](../coordination-standard/README.md).

## Why this standard

As the owner, you watch one thing: the agent list in your main Herdr sidebar, where each agent shows as waiting for you or done. The workers that the coordinator sends out (producers, review seats, QA, and the others) are not your concern. They start often, talk with the coordinator often, and close often. They must not pull your attention away from more important work.

So the workers run in a separate Herdr session named `workers`, and your sidebar does not show them. It shows only the coordinator, the main-line agent. The coordinator tracks the state of every worker it sends out. It interrupts you only when all the work is done, or when it and its workers cannot solve a problem.

![Owner focus](assets/owner_focus.svg)

| File | What it holds | Share it? |
|---|---|---|
| [`coordinator.md`](coordinator.md) | Every rule: roles, the three review lanes, routes, cards, waiting, QA and review, merge and publication. Names no worker model | Yes, portable |
| [`model_bindings.json`](model_bindings.json) | Which model, harness flags, and reasoning effort each route uses by default, and what the coordinator may adjust | Personal; write your own |

This README only explains. If it disagrees with the card, the card wins.

## How it fits together

![System architecture](assets/system_architecture.svg)

The coordinator never writes candidates. Each producer or seat is launched from `model_bindings.json` in a new Tab on its own worktree, inside the `workers` Herdr session. Workers keep running if the coordinator ends. To look at them, run `herdr session attach workers` in another terminal window. Seats review a detached worktree at the frozen commit. Cards, reports, and QA logs live in `<coord>`, outside the repository.

## Lifecycle

![Workflow lifecycle](assets/workflow_lifecycle.svg)

Lanes ([card §2](coordinator.md#2-lanes-and-gates)): ROUTINE is QA plus a coordinator check, ELEVATED adds a REVIEW seat, CRITICAL adds AUDIT and AUDIT_2.

## Getting started

Open a Herdr Tab, start Claude Code there, and point it at the files where they are (don't copy them):

```text
Read <dir>/coordinator.md and <dir>/model_bindings.json as the coordinator policy,
within the owner's authorization and project rules. Declared version: 0.8.0.
Do not load coordination-standard/ or archive/. Follow the card.
```

## Swapping a model

Edit `models.<name>.id`, or a route's `model`, `effort`, or `replacement`, in `model_bindings.json`, and set `updated`. Check flags with `claude --help` and `codex --help` after CLI updates. Nothing else changes, with one caveat: under `-s workspace-write`, a Codex session cannot commit in a linked worktree (its git metadata lives in the main repository), so keep producer routes on the claude harness.

Route bindings are defaults. When the coordinator is highly confident that another model or effort fits a task better, it may change one launch on its own (claude) harness to another model in `models`, with effort from `min_effort` to `max_effort`, and it logs the reason ([card §3](coordinator.md#3-routes-and-launching)). Routes on other harnesses stay as bound. To change what it may adjust, edit `adjustments`.

