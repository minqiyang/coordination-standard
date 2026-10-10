// Command coord-guard is a Claude Code PreToolUse hook for the coordination
// standard. It is a guardrail, not a security boundary.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

const usage = `usage:
  coord-guard hook                               read a PreToolUse event on stdin
  coord-guard qa <task-dir> <name> -- <cmd...>   run QA and log it to <task-dir>/qa/<sha7>-<name>.log
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "hook":
		os.Exit(runHook(os.Stdin, os.Stdout, os.Stderr, os.Getenv))
	case "qa":
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "coord-guard qa:", err)
			os.Exit(125)
		}
		os.Exit(runQA(cwd, os.Args[2:], os.Stdin, os.Stdout, os.Stderr))
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
}

type hookInput struct {
	SessionID string `json:"session_id"`
	Cwd       string `json:"cwd"`
	ToolName  string `json:"tool_name"`
	ToolInput struct {
		Command string `json:"command"`
	} `json:"tool_input"`
}

type hookOutput struct {
	HookSpecificOutput struct {
		HookEventName            string `json:"hookEventName"`
		PermissionDecision       string `json:"permissionDecision"`
		PermissionDecisionReason string `json:"permissionDecisionReason"`
	} `json:"hookSpecificOutput"`
}

// runHook always returns 0. A guard bug must not stop all work, so on any
// internal error it allows the call and warns on stderr.
func runHook(stdin io.Reader, stdout, stderr io.Writer, getenv func(string) string) (code int) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(stderr, "coord-guard: warning: internal error, call allowed: %v\n", r)
			code = 0
		}
	}()
	raw, err := io.ReadAll(stdin)
	if err != nil {
		fmt.Fprintf(stderr, "coord-guard: warning: cannot read hook input, call allowed: %v\n", err)
		return 0
	}
	var in hookInput
	if err := json.Unmarshal(raw, &in); err != nil {
		fmt.Fprintf(stderr, "coord-guard: warning: hook input is not valid JSON, call allowed: %v\n", err)
		return 0
	}
	if in.ToolName != "Bash" || strings.TrimSpace(in.ToolInput.Command) == "" {
		return 0
	}
	cwd := in.Cwd
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	d := Check(in.ToolInput.Command, cwd, getenv)
	if d == nil {
		return 0
	}
	var out hookOutput
	out.HookSpecificOutput.HookEventName = "PreToolUse"
	out.HookSpecificOutput.PermissionDecision = "deny"
	out.HookSpecificOutput.PermissionDecisionReason = d.Reason
	enc := json.NewEncoder(stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(stderr, "coord-guard: warning: cannot write decision: %v\n", err)
	}
	return 0
}

// runQA implements `coord-guard qa <task-dir> <name> -- <cmd...>`. The words
// after -- are joined with spaces and run as one sh -c script. It returns the
// command's exit status, or 125 when it cannot start the QA run.
func runQA(cwd string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fail := func(format string, a ...any) int {
		fmt.Fprintf(stderr, "coord-guard qa: "+format+"\n", a...)
		return 125
	}
	if len(args) < 4 || args[2] != "--" {
		return fail("usage: coord-guard qa <task-dir> <name> -- <cmd...>")
	}
	taskDir, name, argv := args[0], args[1], args[3:]
	if name == "" || strings.ContainsAny(name, "/\\ \t\n") {
		return fail("name %q must be one word with no slash", name)
	}
	if !isDir(taskDir) {
		return fail("task folder %s does not exist", taskDir)
	}
	head, err := git(cwd, "rev-parse", "HEAD")
	if err != nil {
		return fail("cwd %s is not a git worktree: %v", cwd, err)
	}
	if _, err := git(cwd, "symbolic-ref", "-q", "HEAD"); err == nil {
		return fail("cwd %s is on a branch; run QA in a detached worktree at the frozen SHA", cwd)
	}
	if st, err := git(cwd, "status", "--porcelain"); err != nil || st != "" {
		return fail("cwd %s is not clean; run QA in a clean detached worktree", cwd)
	}
	qaDir := filepath.Join(taskDir, "qa")
	if err := os.MkdirAll(qaDir, 0o755); err != nil {
		return fail("%v", err)
	}
	logPath := filepath.Join(qaDir, head[:7]+"-"+name+".log")
	f, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fail("%s already exists; QA logs are never overwritten, so use a new name", logPath)
		}
		return fail("%v", err)
	}
	defer f.Close()

	script := strings.Join(argv, " ")
	lw := &logWriter{w: f}
	fmt.Fprintf(lw, "command: %s\nhead: %s\ncwd: %s\n", script, head, cwd)

	cmd := exec.Command("sh", "-c", script)
	cmd.Dir = cwd
	cmd.Stdin = stdin
	cmd.Stdout = io.MultiWriter(stdout, lw)
	cmd.Stderr = io.MultiWriter(stderr, lw)
	status := 0
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			fmt.Fprintf(lw, "coord-guard qa: cannot run command: %v\n", err)
			status = 127
		} else if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			status = 128 + int(ws.Signal())
		} else {
			status = ee.ExitCode()
		}
	}
	if !lw.endsWithNewline() {
		fmt.Fprintln(lw)
	}
	fmt.Fprintf(lw, "exit status: %d\n", status)
	fmt.Fprintf(stderr, "coord-guard qa: log %s, exit status %d\n", logPath, status)
	return status
}

type logWriter struct {
	mu   sync.Mutex
	w    io.Writer
	last byte
}

func (l *logWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(p) > 0 {
		l.last = p[len(p)-1]
	}
	return l.w.Write(p)
}

func (l *logWriter) endsWithNewline() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.last == '\n'
}
