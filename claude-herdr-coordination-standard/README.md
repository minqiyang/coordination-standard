# Claude-herdr Coordination Standard

**Version `0.4.0`** (derived from Coordination Standard 0.9.0)

A working standard for running several AI coding agents on one project when the coordinator is a **Claude Code** session in a **Herdr** Tab. Producers, review seats, and integrators each run in their own Herdr Tab and git worktree; the coordinator dispatches, waits in the background, verifies, and gates. For any other coordinator, use [Coordination Standard 0.12.0](../coordination-standard/README.md).

| File | What it holds | Share it? |
|---|---|---|
| [`coordinator.md`](coordinator.md) | Every rule: roles, the three review lanes, routes, cards, waiting, QA and review, merge and publication. Names no worker model | Yes, portable |
| [`model_bindings.json`](model_bindings.json) | Which model, harness flags, and reasoning effort each route uses right now | Personal; write your own |

This README only explains. If it disagrees with the card, the card wins.

## How it fits together

![System architecture](assets/system_architecture.svg)

The coordinator never writes candidates. Each producer or seat is launched from `model_bindings.json` in a new Tab on its own worktree; seats review a detached worktree at the frozen commit. Cards, reports, and QA logs live in `<coord>`, outside the repository.

## Lifecycle

![Workflow lifecycle](assets/workflow_lifecycle.svg)

Lanes ([card §2](coordinator.md#2-lanes-and-gates)): ROUTINE is QA plus a coordinator check, ELEVATED adds a REVIEW seat, CRITICAL adds AUDIT and AUDIT_2.

## Getting started

Open a Herdr Tab, start Claude Code there, and point it at the files where they are (don't copy them):

```text
Read <dir>/coordinator.md and <dir>/model_bindings.json as the coordinator policy,
within the owner's authorization and project rules. Declared version: 0.4.0.
Do not load coordination-standard/ or archive/. Follow the card.
```

## Swapping a model

Edit `models.<name>.id`, or a route's `model`, `effort`, or `replacement`, in `model_bindings.json`, and set `updated`. Check flags with `claude --help` and `codex --help` after CLI updates. Nothing else changes, with one caveat: under `-s workspace-write`, a Codex session cannot commit in a linked worktree (its git metadata lives in the main repository), so keep producer routes on the claude harness.

## Differences from the general standard

- The coordinator is Claude Code and judges technically, but never writes candidates and never dismisses a MATERIAL finding.
- The routing table is merged into the card; the bindings file is flat and builds every launch line.
- Five routes: DESIGN and FRONTEND fold into GENERAL_EXEC.
- A background `herdr agent prompt --wait` wakes the coordinator, replacing the polling and wait rules.
- Writers and seats work in git worktrees; seats are checked for read-only behavior after review.
- No Artifact Guard or Contract First skill.
