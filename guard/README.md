# coord-guard

coord-guard is a Claude Code PreToolUse hook for the Claude-herdr coordinator card (`../claude-herdr-coordination-standard/coordinator.md`). It checks Bash tool calls against three rules of the card and denies a call that breaks one. All other rules stay as text in the card.

It is a guardrail, not a security boundary. It stops mistakes. It does not stop a person or agent who wants to get around it.

## What it checks

The guard parses each Bash command with `mvdan.cc/sh/v3/syntax` and checks every simple command in it. This includes commands in lists, pipelines, subshells, command substitutions, and background jobs. It looks through `nohup`, `timeout`, `env`, `nice`, `command`, `exec`, `eval`, and `sh -c` / `bash -c` scripts.

If a word that a rule reads is not literal (a variable or a command substitution), the guard denies the call and asks for a literal command. If the command does not parse, the guard denies it only when the text contains `codex` or `gh pr merge`.

**R1, codex sandbox.** For `codex exec`:

- A new session needs `-s read-only` (or `--sandbox read-only`) after `exec`.
- `codex exec resume` and `codex exec fork` have no `-s` flag. They need `-c sandbox_mode="read-only"`. Without it, they use the config default.
- The guard denies `--dangerously-bypass-approvals-and-sandbox`, `--full-auto`, `--approve-for-me`, `--yolo`, and any sandbox value other than `read-only`, from `-s` or from `-c sandbox_mode=...`.
- Other codex commands, such as `codex --version` and `codex login status`, are allowed.

**R2, merge form and acceptance.** For `gh pr merge`:

- The call needs `--squash` and `--match-head-commit <40-hex SHA>`.
- The only other allowed flags are `-R/--repo`, `--subject`, `--body`, `--body-file`, and `--delete-branch`. The guard denies all other flags, including `--auto`, `--admin`, `--merge`, and `--rebase`.
- Some `<coord>/*/decisions.md` must have a line that starts with `ACCEPT <SHA>`.
- For repos that require an up-to-date branch, the line can be `ACCEPT <H> from <A>`. Then the guard checks, with local git objects only, that H has exactly two parents, A and one other commit B, that `git merge-tree --write-tree A B` has no conflict, and that its tree equals the tree of H. It also requires passing QA for H, with the same rule as R3. The guard never fetches.

**R3, QA before a codex seat.** A seat launch is `codex exec` (not resume or fork) with `-C <worktree>`. The worktree must be detached and clean (`git status --porcelain` is empty). At least one `<coord>/*/qa/<sha7>-*.log` must exist for its HEAD, and every such log must end with the line `exit status: 0`. Evidence logs named `<sha7>-ev-*.log` do not count.

**Finding `<coord>`.** The guard uses `COORD_DIR` if it is set. Otherwise it uses the nearest `*.coord` folder above the hook cwd. Otherwise it finds the main repo root from `git rev-parse --path-format=absolute --git-common-dir` and uses the sibling `<repo>.coord/`. If a rule needs `<coord>` and the guard cannot find it, the guard denies the call.

## What it does not check

- Tools other than Bash, and all rules of the card other than R1 to R3.
- Commands that it cannot see: a script file, a heredoc passed to `sh`, `xargs`, `sudo`, aliases, and shell functions.
- A command name that comes from a variable, unless the word text names `codex` or `gh`.
- A relative `-C` path after a `cd` in the same command. The guard resolves it from the hook cwd.
- Top-level `codex review`, `codex resume`, and `codex fork` (interactive or non-`exec` forms).

If the guard itself fails (bad input, a panic), it allows the call and writes a warning to stderr. A guard bug must not stop all work.

## QA wrapper

```sh
coord-guard qa <task-dir> <name> -- <cmd...>
```

Run it from a clean detached worktree. It joins the words after `--` with spaces and runs them as one `sh -c` script. It writes the command, HEAD, and cwd to `<task-dir>/qa/<sha7>-<name>.log`, copies all output to the log and the terminal, and writes `exit status: <n>` as the last line. It exits with the command's status, or 125 if it cannot start. It never overwrites a log: use a new name for a rerun.

## Build

```sh
make test      # go vet, then go test
make build     # bin/coord-guard
make install   # copies bin/coord-guard to ~/.local/bin/coord-guard
```

## Register

This is an example only. Add it to `~/.claude/settings.json` yourself when you are ready:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          { "type": "command", "command": "$HOME/.local/bin/coord-guard hook", "timeout": 30 }
        ]
      }
    ]
  }
}
```

The hook reads the PreToolUse JSON on stdin. To deny, it prints `{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"..."}}` and exits 0. Claude sees the reason. To allow, it prints nothing and exits 0. Each reason starts with `coord-guard:`, says what to fix, and ends with "A guard denial is a policy stop. Fix the cause; do not route around it."
