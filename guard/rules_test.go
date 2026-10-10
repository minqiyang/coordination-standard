package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ---------- fixtures ----------

type fx struct {
	t                       *testing.T
	root, repo, coord, task string
	wt, sha                 string
	cwd                     string
	env                     map[string]string
}

func gitT(t *testing.T, dir string, args ...string) string {
	t.Helper()
	base := []string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.com",
		"-c", "commit.gpgsign=false", "-c", "init.defaultBranch=main"}
	out, err := exec.Command("git", append(base, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func write(t *testing.T, p, s string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func commit(t *testing.T, dir, file, content, msg string) string {
	t.Helper()
	write(t, filepath.Join(dir, file), content)
	gitT(t, dir, "add", file)
	gitT(t, dir, "commit", "-q", "-m", msg)
	return gitT(t, dir, "rev-parse", "HEAD")
}

func newFx(t *testing.T) *fx {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := &fx{t: t, root: root, repo: filepath.Join(root, "proj"), env: map[string]string{}}
	f.coord = f.repo + ".coord"
	f.task = filepath.Join(f.coord, "task1")
	if err := os.MkdirAll(filepath.Join(f.task, "qa"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(f.repo, 0o755); err != nil {
		t.Fatal(err)
	}
	gitT(t, f.repo, "init", "-q")
	f.sha = commit(t, f.repo, "a.txt", "one\n", "base")
	f.wt = filepath.Join(f.coord, "wt", "review-task1-1")
	gitT(t, f.repo, "worktree", "add", "-q", "--detach", f.wt, f.sha)
	f.cwd = f.repo
	return f
}

func (f *fx) qa(name, body string) {
	write(f.t, filepath.Join(f.task, "qa", f.sha[:7]+"-"+name+".log"), body)
}

func (f *fx) decisions(body string) {
	write(f.t, filepath.Join(f.task, "decisions.md"), body)
}

func (f *fx) expand(s string) string {
	return strings.NewReplacer("{wt}", f.wt, "{coord}", f.coord, "{sha}", f.sha, "{SHA}", testSHA).Replace(s)
}

func (f *fx) check(cmd string) *Decision {
	return Check(f.expand(cmd), f.cwd, func(k string) string { return f.env[k] })
}

type tcase struct {
	name  string
	cmd   string
	setup func(*fx)
	deny  string // "" means allow; else a substring of the reason
}

func run(t *testing.T, cases []tcase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFx(t)
			if tc.setup != nil {
				tc.setup(f)
			}
			d := f.check(tc.cmd)
			switch {
			case tc.deny == "" && d != nil:
				t.Fatalf("want allow, got deny: %s", d.Reason)
			case tc.deny != "" && d == nil:
				t.Fatalf("want deny containing %q, got allow", tc.deny)
			case d != nil:
				if !strings.HasPrefix(d.Reason, "coord-guard: ") || !strings.HasSuffix(d.Reason, denyTail) {
					t.Fatalf("bad reason frame: %s", d.Reason)
				}
				if !strings.Contains(d.Reason, tc.deny) {
					t.Fatalf("reason %q does not contain %q", d.Reason, tc.deny)
				}
			}
		})
	}
}

const pass = "command: make test\nok\nexit status: 0\n"

func qaPass(f *fx) { f.qa("unit", pass) }

// ---------- R1 ----------

func TestR1CodexSandbox(t *testing.T) {
	run(t, []tcase{
		{name: "version", cmd: "codex --version"},
		{name: "login status", cmd: "codex login status"},
		{name: "help", cmd: "codex --help"},
		{name: "exec help", cmd: "codex exec --help"},
		{name: "exec help sub", cmd: "codex exec help resume"},
		{name: "echo mentions codex", cmd: "echo codex exec --full-auto"},
		{name: "read-only no -C", cmd: "codex exec -s read-only -o r.md - < card.md"},
		{name: "long sandbox eq", cmd: "codex exec --sandbox=read-only -o r.md -"},
		{name: "attached short", cmd: "codex exec -sread-only -"},
		{name: "free value quoted var", cmd: `codex exec -s read-only -o "$OUT" -`},
		{name: "resume with override", cmd: `codex exec resume 019a7c3e -m gpt-6.1-sol -c sandbox_mode="read-only" -o r2.md "why?"`},
		{name: "resume quoted override", cmd: `codex exec resume 019a -c 'sandbox_mode = "read-only"' -o r.md q`},
		{name: "resume bare value", cmd: `codex exec resume 019a -c sandbox_mode=read-only q`},
		{name: "resume via root -c", cmd: `codex -c sandbox_mode="read-only" exec resume 019a q`},
		{name: "fork with override", cmd: `codex exec fork 019a -c sandbox_mode="read-only" q`},
		{name: "review read-only", cmd: `codex exec -s read-only review --base main`},

		{name: "incident resume no override", deny: "falls back",
			cmd: `codex exec resume 019a7c3e -m gpt-6.1-sol -c 'model_reasoning_effort="xhigh"' "Answer the follow-up." -o x`},
		{name: "resume workspace-write", deny: "sandbox_mode", cmd: `codex exec resume 019a -c sandbox_mode="workspace-write" q`},
		{name: "fork no override", deny: "falls back", cmd: `codex exec fork 019a q`},
		{name: "new no sandbox", deny: "-s read-only", cmd: "codex exec -o r.md - < card.md"},
		{name: "workspace-write", deny: "workspace-write", cmd: "codex exec -s workspace-write -"},
		{name: "danger-full-access", deny: "danger-full-access", cmd: "codex exec --sandbox danger-full-access -"},
		{name: "bypass", deny: "--dangerously-bypass", cmd: "codex exec -s read-only --dangerously-bypass-approvals-and-sandbox -"},
		{name: "full-auto", deny: "--full-auto", cmd: "codex exec --full-auto -"},
		{name: "approve-for-me", deny: "--approve-for-me", cmd: "codex exec -s read-only --approve-for-me -"},
		{name: "-c overrides -s", deny: "danger-full-access", cmd: `codex exec -s read-only -c sandbox_mode="danger-full-access" -`},
		{name: "alias e", deny: "-s read-only", cmd: "codex e -o r -"},
		{name: "root -s only", deny: "-s read-only", cmd: "codex -s read-only exec -o r -"},
		{name: "full path", deny: "-s read-only", cmd: "/opt/bin/codex exec -o r -"},
		{name: "assign prefix", deny: "-s read-only", cmd: "X=1 codex exec -o r -"},
		{name: "in cmd subst", deny: "-s read-only", cmd: "echo $(codex exec -o r -)"},
		{name: "var sandbox", deny: "not a literal", cmd: `codex exec -s "$MODE" -`},
		{name: "var flags", deny: "not a literal", cmd: `codex exec $FLAGS -s read-only -`},
		{name: "var name", deny: "not a literal", cmd: `"$HOME/.local/bin/codex" exec -s read-only -`},
		{name: "var prompt", deny: "not a literal", cmd: `codex exec -s read-only "$(cat card.md)"`},
		{name: "parse error with codex", deny: "cannot read", cmd: `codex exec 'unterminated`},
		{name: "parse error without codex", cmd: `echo 'unterminated`},
	})
}

// ---------- R2 ----------

var testSHA = strings.Repeat("ab", 20)

func accept(f *fx) { f.decisions("# decisions\n\nACCEPT " + testSHA + "\n") }

func TestR2Merge(t *testing.T) {
	run(t, []tcase{
		{name: "accepted", setup: accept, cmd: "gh pr merge 231 --squash --match-head-commit {SHA}"},
		{name: "allowed extras", setup: accept,
			cmd: `gh pr merge 231 -R o/r --squash --match-head-commit {SHA} --subject "s" --body "$(cat notes.md)" --body-file b.md --delete-branch`},
		{name: "eq form", setup: accept, cmd: "gh pr merge 231 --squash --match-head-commit={SHA}"},
		{name: "short cluster", setup: accept, cmd: "gh pr merge 231 -sd --match-head-commit {SHA}"},
		{name: "repo before merge", setup: accept, cmd: "gh pr -R o/r merge 5 --squash --match-head-commit {SHA}"},
		{name: "accept with note", setup: func(f *fx) { f.decisions("ACCEPT " + testSHA + " gate ELEVATED\n") },
			cmd: "gh pr merge 231 --squash --match-head-commit {SHA}"},
		{name: "from coord cwd", setup: func(f *fx) { accept(f); f.cwd = f.coord },
			cmd: "gh pr merge 231 --squash --match-head-commit {SHA}"},
		{name: "pr view", cmd: "gh pr view 12"},
		{name: "merge help", cmd: "gh pr merge --help"},

		{name: "incident no match-head-commit", setup: accept, deny: "--match-head-commit", cmd: "gh pr merge 281 --squash"},
		{name: "incident no ACCEPT", deny: "no line \"ACCEPT", cmd: "gh pr merge 231 --squash --match-head-commit {SHA}"},
		{name: "incident auto", setup: accept, deny: "--auto", cmd: "gh pr merge 12 --auto --squash"},
		{name: "admin", setup: accept, deny: "--admin", cmd: "gh pr merge 1 --squash --admin --match-head-commit {SHA}"},
		{name: "merge", setup: accept, deny: "--merge", cmd: "gh pr merge 1 --merge --match-head-commit {SHA}"},
		{name: "-m", setup: accept, deny: "-m", cmd: "gh pr merge 1 -m --match-head-commit {SHA}"},
		{name: "rebase", setup: accept, deny: "--rebase", cmd: "gh pr merge 1 --rebase --match-head-commit {SHA}"},
		{name: "-r in cluster", setup: accept, deny: "-r", cmd: "gh pr merge 1 -sr --match-head-commit {SHA}"},
		{name: "unknown flag", setup: accept, deny: "-A", cmd: "gh pr merge 1 --squash --match-head-commit {SHA} -A me@x"},
		{name: "no squash", setup: accept, deny: "--squash", cmd: "gh pr merge 1 --match-head-commit {SHA}"},
		{name: "squash false", setup: accept, deny: "--squash", cmd: "gh pr merge 1 --squash=false --match-head-commit {SHA}"},
		{name: "short sha", setup: accept, deny: "40-hex", cmd: "gh pr merge 1 --squash --match-head-commit abababa"},
		{name: "var sha", setup: accept, deny: "not a literal", cmd: `gh pr merge 1 --squash --match-head-commit "$SHA"`},
		{name: "var subcommand", deny: "not a literal", cmd: `gh $SUB merge 1 --squash`},
		{name: "indented accept", setup: func(f *fx) { f.decisions("  ACCEPT " + testSHA + "\n") },
			deny: "no line", cmd: "gh pr merge 1 --squash --match-head-commit {SHA}"},
		{name: "list accept", setup: func(f *fx) { f.decisions("- ACCEPT " + testSHA + "\n") },
			deny: "no line", cmd: "gh pr merge 1 --squash --match-head-commit {SHA}"},
		{name: "longer token", setup: func(f *fx) { f.decisions("ACCEPT " + testSHA + "ff\n") },
			deny: "no line", cmd: "gh pr merge 1 --squash --match-head-commit {SHA}"},
		{name: "wrong depth", setup: func(f *fx) { write(f.t, filepath.Join(f.task, "sub", "decisions.md"), "ACCEPT "+testSHA+"\n") },
			deny: "no line", cmd: "gh pr merge 1 --squash --match-head-commit {SHA}"},
		{name: "in subshell list", setup: accept, deny: "--auto", cmd: "(cd /tmp && gh pr merge 12 --auto --squash) || true"},
		{name: "no coord", setup: func(f *fx) { f.cwd = f.root }, deny: "cannot find the coord folder",
			cmd: "gh pr merge 1 --squash --match-head-commit {SHA}"},
		{name: "COORD_DIR", setup: func(f *fx) { accept(f); f.cwd = f.root; f.env["COORD_DIR"] = f.coord },
			cmd: "gh pr merge 1 --squash --match-head-commit {SHA}"},
	})
}

// updateFx builds base X, accepted A on a branch, base head B on main, and
// H = merge of B into A. mode "clean", "extra" or "conflict".
func updateFx(t *testing.T, f *fx, mode string) (h, a string) {
	t.Helper()
	gitT(t, f.repo, "checkout", "-q", "-b", "feat")
	if mode == "conflict" {
		a = commit(t, f.repo, "a.txt", "feat\n", "A")
	} else {
		a = commit(t, f.repo, "f.txt", "feat\n", "A")
	}
	gitT(t, f.repo, "checkout", "-q", "main")
	if mode == "conflict" {
		commit(t, f.repo, "a.txt", "main\n", "B")
	} else {
		commit(t, f.repo, "m.txt", "main\n", "B")
	}
	gitT(t, f.repo, "checkout", "-q", "feat")
	if mode == "conflict" {
		cmd := exec.Command("git", "-C", f.repo, "-c", "user.name=t", "-c", "user.email=t@example.com", "merge", "-q", "--no-ff", "-m", "update", "main")
		if err := cmd.Run(); err == nil {
			t.Fatal("expected a merge conflict")
		}
		write(t, filepath.Join(f.repo, "a.txt"), "resolved\n")
		gitT(t, f.repo, "add", "a.txt")
		gitT(t, f.repo, "commit", "-q", "--no-edit")
	} else {
		gitT(t, f.repo, "merge", "-q", "--no-ff", "-m", "update", "main")
	}
	if mode == "extra" {
		write(t, filepath.Join(f.repo, "f.txt"), "feat plus extra\n")
		gitT(t, f.repo, "commit", "-q", "-a", "--amend", "--no-edit")
	}
	return gitT(t, f.repo, "rev-parse", "HEAD"), a
}

func TestR2UpdateMerge(t *testing.T) {
	type c struct {
		name, mode, deny string
		qa               bool
		cwdCoord         bool
		from             func(h, a, x string) string
	}
	std := func(h, a, _ string) string { return "ACCEPT " + h + " from " + a + "\n" }
	for _, tc := range []c{
		{name: "clean update merge", mode: "clean", qa: true, from: std},
		{name: "clean from coord cwd", mode: "clean", qa: true, cwdCoord: true, from: std},
		{name: "extra change", mode: "extra", qa: true, from: std, deny: "differs from the clean merge"},
		{name: "conflicted merge", mode: "conflict", qa: true, from: std, deny: "has conflicts"},
		{name: "no QA for H", mode: "clean", from: std, deny: "no QA log"},
		{name: "from is not a parent", mode: "clean", qa: true, deny: "neither parent",
			from: func(h, _, x string) string { return "ACCEPT " + h + " from " + x + "\n" }},
		{name: "from missing locally", mode: "clean", qa: true, deny: "not in the local repo",
			from: func(h, _, _ string) string { return "ACCEPT " + h + " from " + strings.Repeat("cd", 20) + "\n" }},
		{name: "single parent", mode: "single", qa: true, deny: "parent(s), not 2",
			from: func(h, a, _ string) string { return "ACCEPT " + h + " from " + a + "\n" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFx(t)
			var h, a string
			if tc.mode == "single" {
				a = f.sha
				h = commit(t, f.repo, "s.txt", "x\n", "one parent")
			} else {
				h, a = updateFx(t, f, tc.mode)
			}
			f.decisions(tc.from(h, a, f.sha))
			if tc.qa {
				write(t, filepath.Join(f.task, "qa", h[:7]+"-unit.log"), pass)
			}
			if tc.cwdCoord {
				f.cwd = f.coord
			}
			d := f.check("gh pr merge 7 --squash --match-head-commit " + h)
			if tc.deny == "" && d != nil {
				t.Fatalf("want allow, got %s", d.Reason)
			}
			if tc.deny != "" && (d == nil || !strings.Contains(d.Reason, tc.deny)) {
				t.Fatalf("want deny containing %q, got %+v", tc.deny, d)
			}
		})
	}
}

// ---------- R3 ----------

const seat = `timeout 3600 codex exec -m gpt-6.1-sol -c 'model_reasoning_effort="xhigh"' -s read-only -C {wt} -o {coord}/task1/review-task1-1/1/report.md - < card.md`

func TestR3SeatQA(t *testing.T) {
	nohup := `(nohup sh -c "timeout 3600 codex exec -m gpt-6.1-sol -c 'model_reasoning_effort=\"xhigh\"' -s read-only -C {wt} -o {coord}/r.md - < card.md > /dev/null 2> seat.log" &)`
	run(t, []tcase{
		{name: "qa passed", setup: qaPass, cmd: seat},
		{name: "evidence log ignored", setup: func(f *fx) { qaPass(f); f.qa("ev-run", "boom\n") }, cmd: seat},
		{name: "cwd in coord", setup: func(f *fx) { qaPass(f); f.cwd = f.task }, cmd: seat},
		{name: "relative -C from coord", setup: func(f *fx) { qaPass(f); f.cwd = f.coord },
			cmd: "codex exec -s read-only -C wt/review-task1-1 -o r.md - < card.md"},
		{name: "resume skips R3", cmd: `cd {wt} && codex exec resume 019a -c sandbox_mode="read-only" -o r.md q`},

		{name: "incident no exit status", setup: func(f *fx) { f.qa("unit", "command: make test\nok\n") }, deny: "does not end", cmd: seat},
		{name: "incident QA not run", deny: "no QA log", cmd: seat},
		{name: "exit status 1", setup: func(f *fx) { f.qa("unit", "x\nexit status: 1\n") }, deny: "exit status: 1", cmd: seat},
		{name: "one of two fails", setup: func(f *fx) { qaPass(f); f.qa("lint", "exit status: 2\n") }, deny: "lint", cmd: seat},
		{name: "only evidence logs", setup: func(f *fx) { f.qa("ev-run", pass) }, deny: "no QA log", cmd: seat},
		{name: "dirty worktree", setup: func(f *fx) { qaPass(f); write(f.t, filepath.Join(f.wt, "new.txt"), "x") }, deny: "uncommitted", cmd: seat},
		{name: "branch worktree", deny: "on a branch", cmd: seat, setup: func(f *fx) {
			qaPass(f)
			f.wt = filepath.Join(f.coord, "wt", "prod-task1-1")
			gitT(f.t, f.repo, "worktree", "add", "-q", "-b", "task1", f.wt, f.sha)
		}},
		{name: "not a worktree", setup: qaPass, deny: "cannot read HEAD", cmd: "codex exec -s read-only -C {coord}/nowhere -"},

		{name: "nohup sh -c with qa", setup: qaPass, cmd: nohup},
		{name: "nohup sh -c no qa", deny: "no QA log", cmd: nohup},
		{name: "env timeout no qa", deny: "no QA log",
			cmd: "env FOO=1 BAR=2 timeout -k 5 3600 codex exec -s read-only -C {wt} -o r.md - < card.md"},
		{name: "bash -lc no qa", deny: "no QA log",
			cmd: "cd {wt} && bash -lc 'codex exec -c model_reasoning_effort=xhigh -s read-only -C {wt} -o r.md - < card.md'"},
		{name: "pipeline no qa", deny: "no QA log", cmd: "cat card.md | nohup codex exec -s read-only -C {wt} -o r.md -"},
		{name: "eval no qa", deny: "no QA log", cmd: `eval "codex exec -s read-only -C {wt} -o r.md -"`},
		{name: "after list no qa", deny: "no QA log", cmd: "true; false || codex exec -s read-only -C {wt} -o r.md - &"},
		{name: "var worktree", setup: qaPass, deny: "not a literal",
			cmd: `WT={wt}; timeout 3600 codex exec -s read-only -C "$WT" -o r.md - < card.md`},
		{name: "var worktree in sh -c", setup: qaPass, deny: "not a literal",
			cmd: `(nohup sh -c "codex exec -s read-only -C $WT -o r.md - < card.md" &)`},
		{name: "no coord", setup: func(f *fx) { qaPass(f); f.cwd = f.root }, deny: "coord folder",
			cmd: "codex exec -s read-only -C {wt} -o r.md -"},
	})
}

// ---------- hook protocol ----------

func hookJSON(tool, cmd, cwd string) string {
	b, _ := json.Marshal(map[string]any{
		"session_id": "s1", "cwd": cwd, "hook_event_name": "PreToolUse",
		"tool_name": tool, "tool_input": map[string]any{"command": cmd},
	})
	return string(b)
}

func TestHookProtocol(t *testing.T) {
	f := newFx(t)
	noenv := func(string) string { return "" }

	var out, errb bytes.Buffer
	code := runHook(strings.NewReader(hookJSON("Bash", "gh pr merge 12 --auto --squash", f.repo)), &out, &errb, noenv)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	var got hookOutput
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("output %q: %v", out.String(), err)
	}
	h := got.HookSpecificOutput
	if h.HookEventName != "PreToolUse" || h.PermissionDecision != "deny" ||
		!strings.HasPrefix(h.PermissionDecisionReason, "coord-guard:") ||
		!strings.HasSuffix(h.PermissionDecisionReason, "Fix the cause; do not route around it.") {
		t.Fatalf("bad output %+v", h)
	}

	for _, in := range []string{
		hookJSON("Bash", "ls -la", f.repo),
		hookJSON("Write", "codex exec --full-auto", f.repo),
		"not json",
		"",
	} {
		out.Reset()
		errb.Reset()
		if code := runHook(strings.NewReader(in), &out, &errb, noenv); code != 0 || out.Len() != 0 {
			t.Fatalf("input %q: code %d out %q", in, code, out.String())
		}
	}
	if !strings.Contains(errb.String(), "warning") {
		t.Fatalf("want a warning for empty input, got %q", errb.String())
	}
}

// ---------- qa wrapper ----------

func TestQAWrapper(t *testing.T) {
	f := newFx(t)
	var out, errb bytes.Buffer
	qa := func(cwd string, args ...string) int {
		out.Reset()
		errb.Reset()
		return runQA(cwd, args, strings.NewReader(""), &out, &errb)
	}
	read := func(name string) []string {
		b, err := os.ReadFile(filepath.Join(f.task, "qa", f.sha[:7]+"-"+name+".log"))
		if err != nil {
			t.Fatal(err)
		}
		return strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	}

	if code := qa(f.wt, f.task, "unit", "--", "echo hi;", "exit 3"); code != 3 {
		t.Fatalf("exit %d, stderr %s", code, errb.String())
	}
	lines := read("unit")
	if lines[0] != "command: echo hi; exit 3" || lines[len(lines)-1] != "exit status: 3" || !strings.Contains(out.String(), "hi") {
		t.Fatalf("bad log %q", lines)
	}
	if d := f.check(seat); d == nil || !strings.Contains(d.Reason, "exit status: 3") {
		t.Fatalf("want deny for failed QA, got %+v", d)
	}

	f2 := newFx(t)
	if code := qa(f2.wt, f2.task, "smoke", "--", "printf", "no-newline"); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	b, _ := os.ReadFile(filepath.Join(f2.task, "qa", f2.sha[:7]+"-smoke.log"))
	if !strings.HasSuffix(string(b), "no-newline\nexit status: 0\n") {
		t.Fatalf("bad log %q", b)
	}
	if d := f2.check(seat); d != nil {
		t.Fatalf("want allow after passing QA, got %s", d.Reason)
	}

	if code := qa(f2.wt, f2.task, "smoke", "--", "true"); code != 125 {
		t.Fatalf("existing log: want 125, got %d", code)
	}
	if code := qa(f2.wt, f2.task, "x"); code != 125 {
		t.Fatalf("usage: want 125, got %d", code)
	}
	if code := qa(f2.repo, f2.task, "onbranch", "--", "true"); code != 125 {
		t.Fatalf("branch cwd: want 125, got %d", code)
	}
	write(t, filepath.Join(f2.wt, "dirt"), "x")
	if code := qa(f2.wt, f2.task, "dirty", "--", "true"); code != 125 {
		t.Fatalf("dirty cwd: want 125, got %d", code)
	}
}
