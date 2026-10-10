package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/syntax"
)

const denyTail = " A guard denial is a policy stop. Fix the cause; do not route around it."

// Decision is the result of a check. A nil *Decision means allow.
type Decision struct {
	Reason string
}

func deny(format string, a ...any) *Decision {
	return &Decision{Reason: "coord-guard: " + fmt.Sprintf(format, a...) + denyTail}
}

var sha40 = regexp.MustCompile(`^[0-9a-f]{40}$`)

const maxDepth = 4

type checker struct {
	cwd    string
	getenv func(string) string
	home   string

	coord     string
	coordErr  error
	coordDone bool
}

// Check decides on one Bash command. cwd is the hook input cwd.
func Check(command, cwd string, getenv func(string) string) *Decision {
	home := getenv("HOME")
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	c := &checker{cwd: cwd, getenv: getenv, home: home}
	return c.checkScript(command, 0)
}

func (c *checker) checkScript(src string, depth int) *Decision {
	f, err := syntax.NewParser().Parse(strings.NewReader(src), "")
	if err != nil {
		if strings.Contains(src, "codex") || strings.Contains(src, "gh pr merge") {
			return deny("the shell parser cannot read this command (%v), and it mentions codex or gh pr merge. Write it as plain shell syntax.", err)
		}
		return nil
	}
	var d *Decision
	syntax.Walk(f, func(n syntax.Node) bool {
		if d != nil {
			return false
		}
		if ce, ok := n.(*syntax.CallExpr); ok {
			d = c.checkCall(ce.Args, depth)
		}
		return d == nil
	})
	return d
}

// word is one shell word after static resolution.
type word struct {
	s         string // value, valid only when lit is true
	lit       bool   // no parameter, command, arithmetic or process expansion
	splitSafe bool   // every expansion sits inside double quotes
	src       string // source text
}

func (c *checker) resolve(w *syntax.Word) word {
	var buf bytes.Buffer
	syntax.NewPrinter().Print(&buf, w)
	r := word{src: buf.String(), lit: true, splitSafe: true}
	for _, p := range w.Parts {
		switch p.(type) {
		case *syntax.Lit, *syntax.SglQuoted:
		case *syntax.DblQuoted:
			for _, q := range p.(*syntax.DblQuoted).Parts {
				if _, ok := q.(*syntax.Lit); !ok {
					r.lit = false
				}
			}
		default:
			r.lit = false
			r.splitSafe = false
		}
	}
	if r.lit {
		cfg := &expand.Config{Env: expand.ListEnviron("HOME=" + c.home)}
		s, err := expand.Literal(cfg, w)
		if err != nil {
			r.lit = false
		} else {
			r.s = s
		}
	}
	return r
}

func nonLiteral(w word, what string) *Decision {
	return deny("%s is not a literal word: %s. Write the command with literal values (no variables or command substitution) so the guard can check it.", what, w.src)
}

// isAssign reports whether w looks like NAME=VALUE.
var (
	assignRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
	ghName   = regexp.MustCompile(`(^|/)gh"?$`)
)

func isAssign(w *syntax.Word) bool {
	if len(w.Parts) == 0 {
		return false
	}
	l, ok := w.Parts[0].(*syntax.Lit)
	return ok && assignRe.MatchString(l.Value)
}

func (c *checker) checkCall(args []*syntax.Word, depth int) *Decision {
	for len(args) > 0 {
		name := c.resolve(args[0])
		if !name.lit {
			if strings.Contains(name.src, "codex") || ghName.MatchString(name.src) {
				return nonLiteral(name, "the command name")
			}
			return nil
		}
		rest := args[1:]
		switch path.Base(name.s) {
		case "nohup":
			args = rest
		case "command", "exec":
			args = skipOpts(c, rest, map[string]bool{"-a": true})
		case "nice":
			args = skipOpts(c, rest, map[string]bool{"-n": true})
		case "timeout", "gtimeout":
			rest = skipOpts(c, rest, map[string]bool{"-s": true, "-k": true})
			if len(rest) > 0 {
				rest = rest[1:] // duration
			}
			args = rest
		case "env":
			var d *Decision
			args, d = c.skipEnv(rest, depth)
			if d != nil || args == nil {
				return d
			}
		case "sh", "bash", "zsh", "dash", "ksh":
			return c.checkShellC(rest, depth)
		case "eval":
			return c.checkJoined(rest, depth, "eval", "")
		case "codex":
			return c.checkCodex(rest)
		case "gh":
			return c.checkGh(rest)
		default:
			return nil
		}
	}
	return nil
}

// skipOpts drops leading options. Options in withVal take the next word.
func skipOpts(c *checker, ws []*syntax.Word, withVal map[string]bool) []*syntax.Word {
	for len(ws) > 0 {
		w := c.resolve(ws[0])
		if !w.lit || !strings.HasPrefix(w.s, "-") || w.s == "-" {
			return ws
		}
		ws = ws[1:]
		if w.s == "--" {
			return ws
		}
		if withVal[w.s] && len(ws) > 0 {
			ws = ws[1:]
		}
	}
	return ws
}

func (c *checker) skipEnv(ws []*syntax.Word, depth int) ([]*syntax.Word, *Decision) {
	for len(ws) > 0 {
		if isAssign(ws[0]) {
			ws = ws[1:]
			continue
		}
		w := c.resolve(ws[0])
		if !w.lit || !strings.HasPrefix(w.s, "-") || w.s == "-" {
			return ws, nil
		}
		ws = ws[1:]
		switch {
		case w.s == "--":
			return ws, nil
		case w.s == "-S" || w.s == "--split-string":
			return nil, c.checkJoined(ws, depth, "env -S", "")
		case strings.HasPrefix(w.s, "--split-string="):
			return nil, c.checkJoined(ws, depth, "env -S", strings.TrimPrefix(w.s, "--split-string="))
		case w.s == "-u" || w.s == "-C" || w.s == "--unset" || w.s == "--chdir":
			if len(ws) > 0 {
				ws = ws[1:]
			}
		}
	}
	return ws, nil
}

// checkShellC handles `sh -c SCRIPT` and friends by parsing SCRIPT.
func (c *checker) checkShellC(ws []*syntax.Word, depth int) *Decision {
	hasC := false
	for i := 0; i < len(ws); i++ {
		w := c.resolve(ws[i])
		if !hasC {
			if !w.lit || !strings.HasPrefix(w.s, "-") && !strings.HasPrefix(w.s, "+") {
				return nil // a script file, not -c
			}
			if w.s == "-o" || w.s == "+o" {
				i++
				continue
			}
			if !strings.HasPrefix(w.s, "--") && strings.Contains(w.s, "c") {
				hasC = true
			}
			continue
		}
		if !w.lit {
			if strings.Contains(w.src, "codex") || strings.Contains(w.src, "gh ") {
				return nonLiteral(w, "the shell -c script")
			}
			return nil
		}
		return c.nested(w.s, depth)
	}
	return nil
}

func (c *checker) checkJoined(ws []*syntax.Word, depth int, what, first string) *Decision {
	var parts []string
	if first != "" {
		parts = append(parts, first)
	}
	for _, x := range ws {
		w := c.resolve(x)
		if !w.lit {
			if strings.Contains(w.src, "codex") || strings.Contains(w.src, "gh") {
				return nonLiteral(w, "an argument of "+what)
			}
			return nil
		}
		parts = append(parts, w.s)
	}
	return c.nested(strings.Join(parts, " "), depth)
}

func (c *checker) nested(src string, depth int) *Decision {
	if depth >= maxDepth {
		if strings.Contains(src, "codex") || strings.Contains(src, "gh pr merge") {
			return deny("the command nests shells more than %d levels deep. Run codex or gh pr merge with less nesting.", maxDepth)
		}
		return nil
	}
	return c.checkScript(src, depth+1)
}

// ---------- shared option scanning ----------

type opt struct {
	name  string // "-s" or "--sandbox"
	val   word
	level int
}

type scan struct {
	opts []opt
	pos  []word // positional words
}

// scanArgs reads clap or cobra style options. matters(name) tells whether a
// non-literal value of that option must be denied. A free value may be
// non-literal only inside double quotes. A bare non-literal word is denied.
func (c *checker) scanArgs(ws []*syntax.Word, level int, withVal, matters func(string) bool, stopAtPos bool) (scan, int, *Decision) {
	var s scan
	i := 0
	for ; i < len(ws); i++ {
		w := c.resolve(ws[i])
		if !w.lit {
			// Even a quoted expansion can expand to an option.
			return s, i, nonLiteral(w, "an argument")
		}
		if w.s == "--" {
			for _, x := range ws[i+1:] {
				s.pos = append(s.pos, c.resolve(x))
			}
			return s, len(ws), nil
		}
		if !strings.HasPrefix(w.s, "-") || w.s == "-" {
			if stopAtPos {
				return s, i, nil
			}
			s.pos = append(s.pos, w)
			continue
		}
		var name, val string
		hasVal := false
		if strings.HasPrefix(w.s, "--") {
			name, val, hasVal = strings.Cut(w.s, "=")
		} else {
			name = w.s[:2]
			if len(w.s) > 2 {
				if withVal(name) {
					val, hasVal = strings.TrimPrefix(w.s[2:], "="), true
				} else {
					// cluster of boolean short flags, such as -sd
					for _, ch := range w.s[1:] {
						n := "-" + string(ch)
						if withVal(n) {
							return s, i, deny("the short option cluster %s mixes a value option; write each option on its own.", w.s)
						}
						s.opts = append(s.opts, opt{name: n, level: level, val: word{lit: true, splitSafe: true}})
					}
					continue
				}
			}
		}
		o := opt{name: name, level: level}
		if hasVal {
			o.val = word{s: val, lit: true, splitSafe: true, src: val}
		} else if withVal(name) {
			if i+1 >= len(ws) {
				return s, i, deny("option %s has no value.", name)
			}
			i++
			o.val = c.resolve(ws[i])
			if !o.val.lit && (matters(name) || !o.val.splitSafe) {
				return s, i, nonLiteral(o.val, "the value of "+name)
			}
		}
		s.opts = append(s.opts, o)
	}
	return s, i, nil
}

// ---------- R1 and R3: codex ----------

var codexVal = map[string]bool{
	"-c": true, "--config": true, "--enable": true, "--disable": true,
	"--remote": true, "--remote-auth-token-env": true, "-i": true, "--image": true,
	"-m": true, "--model": true, "--local-provider": true, "-p": true, "--profile": true,
	"-s": true, "--sandbox": true, "-C": true, "--cd": true, "--add-dir": true,
	"-a": true, "--ask-for-approval": true, "--thread-source": true,
	"--cyber-access-program": true, "--output-schema": true, "--color": true,
	"-o": true, "--output-last-message": true, "--base": true, "--commit": true, "--title": true,
}

// codexFree lists options whose value no rule reads.
var codexFree = map[string]bool{
	"-o": true, "--output-last-message": true, "-m": true, "--model": true,
	"--output-schema": true, "-i": true, "--image": true, "--title": true,
}

var codexBad = map[string]string{
	"--dangerously-bypass-approvals-and-sandbox": "turns off the sandbox",
	"--full-auto":      "allows workspace writes",
	"--approve-for-me": "uses the workspace-write sandbox",
	"--yolo":           "turns off the sandbox",
}

func (c *checker) checkCodex(ws []*syntax.Word) *Decision {
	withVal := func(n string) bool { return codexVal[n] }
	matters := func(n string) bool { return !codexFree[n] }

	top, i, d := c.scanArgs(ws, 0, withVal, matters, true)
	if d != nil {
		return d
	}
	if i >= len(ws) {
		return nil // no subcommand: interactive codex, --version, --help
	}
	sub := c.resolve(ws[i])
	if sub.s != "exec" && sub.s != "e" {
		return nil
	}
	rest := ws[i+1:]
	lvl1, j, d := c.scanArgs(rest, 1, withVal, matters, true)
	if d != nil {
		return d
	}
	mode := ""
	if j < len(rest) {
		switch p := c.resolve(rest[j]).s; p {
		case "resume", "fork", "review", "help":
			mode = p
			rest = rest[j+1:]
		default:
			rest = rest[j:]
		}
	} else {
		rest = nil
	}
	lvl2, _, d := c.scanArgs(rest, 2, withVal, matters, false)
	if d != nil {
		return d
	}
	if mode == "help" {
		return nil
	}
	all := append(append(top.opts, lvl1.opts...), lvl2.opts...)

	var sOK, cOK bool
	var cds []string
	for _, o := range all {
		switch o.name {
		case "-h", "--help", "-V", "--version":
			return nil
		}
	}
	for _, o := range all {
		if why, bad := codexBad[o.name]; bad {
			return deny("codex exec with %s %s. Seats run read-only: remove it and use -s read-only (or -c sandbox_mode=\"read-only\" on resume).", o.name, why)
		}
		switch o.name {
		case "-s", "--sandbox":
			if o.val.s != "read-only" {
				return deny("codex exec uses sandbox %q. Only read-only is allowed: use -s read-only.", o.val.s)
			}
			if o.level >= 1 {
				sOK = true
			}
		case "-c", "--config":
			k, v, ok := strings.Cut(o.val.s, "=")
			if !ok || strings.TrimSpace(k) != "sandbox_mode" {
				continue
			}
			if tomlString(v) != "read-only" {
				return deny("codex exec sets sandbox_mode to %s. Only read-only is allowed: use -c sandbox_mode=\"read-only\".", strings.TrimSpace(v))
			}
			cOK = true
		case "-C", "--cd":
			cds = append(cds, o.val.s)
		}
	}
	switch mode {
	case "resume", "fork":
		if !cOK {
			return deny("codex exec %s has no sandbox override, so it falls back to the config default (it has run as danger-full-access). Add -c sandbox_mode=\"read-only\".", mode)
		}
		return nil
	default:
		if !sOK {
			return deny("a new codex exec session needs an explicit read-only sandbox. Add -s read-only after exec.")
		}
	}
	for _, wt := range cds {
		if d := c.checkSeatQA(wt); d != nil {
			return d
		}
	}
	return nil
}

func tomlString(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 && (v[0] == '"' && v[len(v)-1] == '"' || v[0] == '\'' && v[len(v)-1] == '\'') {
		return v[1 : len(v)-1]
	}
	return v
}

// checkSeatQA is R3: the seat worktree is clean, detached, and has passing QA.
func (c *checker) checkSeatQA(wt string) *Decision {
	if !filepath.IsAbs(wt) {
		wt = filepath.Join(c.cwd, wt)
	}
	head, err := git(wt, "rev-parse", "HEAD")
	if err != nil {
		return deny("cannot read HEAD of the seat worktree %s (%v). Launch the seat with -C set to a detached worktree at the frozen SHA.", wt, err)
	}
	if _, err := git(wt, "symbolic-ref", "-q", "HEAD"); err == nil {
		return deny("the seat worktree %s is on a branch, not detached. Create it with git worktree add --detach <path> <SHA>.", wt)
	}
	st, err := git(wt, "status", "--porcelain")
	if err != nil {
		return deny("cannot read git status of %s (%v). Fix the worktree, then launch the seat.", wt, err)
	}
	if st != "" {
		return deny("the seat worktree %s has uncommitted or untracked files. Seats review a clean frozen SHA: clean the worktree or make a new detached one.", wt)
	}
	coord, d := c.coordDir()
	if d != nil {
		return d
	}
	if d := qaPassed(coord, head, "the seat launch"); d != nil {
		return d
	}
	return nil
}

// qaPassed checks <coord>/*/qa/<sha7>-*.log, skipping <sha7>-ev-*.log.
func qaPassed(coord, sha, what string) *Decision {
	sha7 := sha[:7]
	logs, _ := filepath.Glob(filepath.Join(coord, "*", "qa", sha7+"-*.log"))
	n := 0
	for _, l := range logs {
		if strings.HasPrefix(filepath.Base(l), sha7+"-ev-") {
			continue
		}
		n++
		last, err := lastLine(l)
		if err != nil {
			return deny("cannot read QA log %s (%v). Fix or rerun that QA before %s.", l, err, what)
		}
		if last != "exit status: 0" {
			return deny("QA log %s does not end with \"exit status: 0\" (last line: %q). QA must pass on %s before %s: fix the cause and rerun QA with coord-guard qa.", l, last, sha7, what)
		}
	}
	if n == 0 {
		return deny("no QA log for %s in %s/*/qa/%s-*.log. Run the required QA first, for example coord-guard qa <task-dir> <name> -- <cmd>, in a clean detached worktree at that SHA.", sha7, coord, sha7)
	}
	return nil
}

func lastLine(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	const tail = 64 << 10
	if fi, err := f.Stat(); err == nil && fi.Size() > tail {
		if _, err := f.Seek(-tail, io.SeekEnd); err != nil {
			return "", err
		}
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(b), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if s := strings.TrimSpace(lines[i]); s != "" {
			return s, nil
		}
	}
	return "", nil
}

// ---------- R2: gh pr merge ----------

var ghVal = map[string]bool{
	"-R": true, "--repo": true, "-t": true, "--subject": true, "-b": true, "--body": true,
	"-F": true, "--body-file": true, "--match-head-commit": true, "-A": true, "--author-email": true,
}

var ghAllowed = map[string]string{
	"-s": "--squash", "--squash": "--squash", "-d": "--delete-branch", "--delete-branch": "--delete-branch",
	"-R": "--repo", "--repo": "--repo", "-t": "--subject", "--subject": "--subject",
	"-b": "--body", "--body": "--body", "-F": "--body-file", "--body-file": "--body-file",
	"--match-head-commit": "--match-head-commit",
}

func (c *checker) checkGh(ws []*syntax.Word) *Decision {
	if len(ws) == 0 {
		return nil
	}
	first := c.resolve(ws[0])
	if !first.lit {
		return nonLiteral(first, "the gh subcommand")
	}
	if first.s != "pr" {
		return nil
	}
	rest := ws[1:]
	// cobra accepts the inherited -R before the subcommand
	for len(rest) > 0 {
		w := c.resolve(rest[0])
		if !w.lit {
			return nonLiteral(w, "the gh pr subcommand")
		}
		if w.s == "-R" || w.s == "--repo" {
			rest = rest[min(2, len(rest)):]
			continue
		}
		if strings.HasPrefix(w.s, "-R") || strings.HasPrefix(w.s, "--repo=") {
			rest = rest[1:]
			continue
		}
		break
	}
	if len(rest) == 0 || c.resolve(rest[0]).s != "merge" {
		return nil
	}
	withVal := func(n string) bool { return ghVal[n] }
	matters := func(n string) bool { return n == "--match-head-commit" }
	s, _, d := c.scanArgs(rest[1:], 0, withVal, matters, false)
	if d != nil {
		return d
	}
	squash := false
	sha := ""
	for _, o := range s.opts {
		if o.name == "--help" || o.name == "-h" {
			return nil
		}
	}
	for _, o := range s.opts {
		canon, ok := ghAllowed[o.name]
		if !ok {
			return deny("gh pr merge with %s is not allowed. Merge with gh pr merge <pr> --squash --match-head-commit <accepted SHA>; only -R/--repo, --subject, --body, --body-file and --delete-branch may be added (no --auto, --admin, --merge or --rebase).", o.name)
		}
		switch canon {
		case "--squash":
			squash = o.val.s == "" || o.val.s == "true"
		case "--match-head-commit":
			sha = strings.ToLower(o.val.s)
		}
	}
	if !squash {
		return deny("gh pr merge needs --squash. Use gh pr merge <pr> --squash --match-head-commit <accepted SHA>.")
	}
	if sha == "" {
		return deny("gh pr merge needs --match-head-commit <40-hex SHA> so the merge binds to the accepted identity. Add it with the full accepted SHA.")
	}
	if !sha40.MatchString(sha) {
		return deny("--match-head-commit value %q is not a full 40-hex SHA. Use the full accepted SHA.", sha)
	}
	coord, d := c.coordDir()
	if d != nil {
		return d
	}
	plain, froms := findAccept(coord, sha)
	if plain {
		return nil
	}
	if len(froms) == 0 {
		return deny("no line \"ACCEPT %s\" at line start in %s/*/decisions.md. Accept that exact SHA through its gate and record it before you merge.", sha, coord)
	}
	var last *Decision
	for _, a := range froms {
		if last = c.checkUpdateMerge(coord, sha, a); last == nil {
			return nil
		}
	}
	return last
}

// findAccept scans <coord>/*/decisions.md for "ACCEPT <sha>" (plain) and
// "ACCEPT <sha> from <A>" lines.
func findAccept(coord, sha string) (plain bool, froms []string) {
	files, _ := filepath.Glob(filepath.Join(coord, "*", "decisions.md"))
	for _, p := range files {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 64<<10), 1<<20)
		for sc.Scan() {
			line := sc.Text()
			if !strings.HasPrefix(line, "ACCEPT ") {
				continue
			}
			fs := strings.Fields(line)
			if len(fs) < 2 || strings.ToLower(fs[1]) != sha {
				continue
			}
			if len(fs) >= 3 && fs[2] == "from" {
				if len(fs) >= 4 && sha40.MatchString(strings.ToLower(fs[3])) {
					froms = append(froms, strings.ToLower(fs[3]))
				}
				continue
			}
			plain = true
		}
		f.Close()
	}
	return plain, froms
}

// checkUpdateMerge decides on merging H when decisions.md has
// "ACCEPT H from A". This is for repos that require an up-to-date branch:
// H must be a clean merge of the base head B into the accepted SHA A, with no
// other change, and QA must have passed on H. The guard uses only local
// objects and never fetches.
func (c *checker) checkUpdateMerge(coord, h, a string) *Decision {
	repo := c.repoDir(coord)
	if repo == "" {
		return deny("cannot find the git repository to verify \"ACCEPT %s from %s\" (hook cwd %s is not in a repo, and %s has no sibling repo). Run the merge from the repo or its coord folder.", h, a, c.cwd, coord)
	}
	for _, x := range []string{h, a} {
		if _, err := git(repo, "cat-file", "-e", x+"^{commit}"); err != nil {
			return deny("commit %s is not in the local repo %s. Fetch it yourself (the guard never fetches), then retry the merge.", x, repo)
		}
	}
	out, err := git(repo, "rev-list", "--parents", "-n1", h)
	if err != nil {
		return deny("cannot read the parents of %s (%v).", h, err)
	}
	fs := strings.Fields(out)
	if len(fs) != 3 {
		return deny("%s has %d parent(s), not 2. An update merge must be a merge of the base head into the accepted SHA %s and nothing else.", h, len(fs)-1, a)
	}
	var b string
	switch a {
	case fs[1]:
		b = fs[2]
	case fs[2]:
		b = fs[1]
	default:
		return deny("neither parent of %s is the accepted SHA %s. Merge the base head into the accepted SHA, or accept %s through its gate.", h, a, h)
	}
	tree, err := git(repo, "merge-tree", "--write-tree", a, b)
	if err != nil {
		return deny("merging base %s into accepted %s has conflicts (git merge-tree --write-tree failed). A conflicted update is a new candidate: send it to a worker and gate it.", b, a)
	}
	tree = strings.SplitN(tree, "\n", 2)[0]
	htree, err := git(repo, "rev-parse", h+"^{tree}")
	if err != nil {
		return deny("cannot read the tree of %s (%v).", h, err)
	}
	if tree != htree {
		return deny("the tree of %s differs from the clean merge of %s and %s, so the update merge carries other changes. Remake the merge with no extra change, or gate %s as a new candidate.", h, a, b, h)
	}
	return qaPassed(coord, h, "the merge of "+h[:7])
}

// repoDir finds a git directory for local object checks.
func (c *checker) repoDir(coord string) string {
	if _, err := git(c.cwd, "rev-parse", "--git-dir"); err == nil {
		return c.cwd
	}
	sib := strings.TrimSuffix(coord, ".coord")
	if sib != coord {
		if _, err := git(sib, "rev-parse", "--git-dir"); err == nil {
			return sib
		}
	}
	return ""
}

// ---------- coord and git ----------

func (c *checker) coordDir() (string, *Decision) {
	if !c.coordDone {
		c.coordDone = true
		c.coord, c.coordErr = findCoord(c.cwd, c.getenv("COORD_DIR"))
	}
	if c.coordErr != nil {
		return "", deny("cannot find the coord folder: %v. Set COORD_DIR, or run from the repo or its <repo>.coord folder.", c.coordErr)
	}
	return c.coord, nil
}

func findCoord(cwd, env string) (string, error) {
	if env != "" {
		if !isDir(env) {
			return "", fmt.Errorf("COORD_DIR=%s is not a folder", env)
		}
		return env, nil
	}
	for d := filepath.Clean(cwd); ; d = filepath.Dir(d) {
		if strings.HasSuffix(filepath.Base(d), ".coord") && isDir(d) {
			return d, nil
		}
		if filepath.Dir(d) == d {
			break
		}
	}
	common, err := git(cwd, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", fmt.Errorf("hook cwd %s is not under a *.coord folder or in a git repo", cwd)
	}
	root := filepath.Dir(common)
	coord := filepath.Join(filepath.Dir(root), filepath.Base(root)+".coord")
	if !isDir(coord) {
		return "", fmt.Errorf("%s does not exist", coord)
	}
	return coord, nil
}

func isDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func git(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			return "", err
		}
		return "", errors.New(msg)
	}
	return strings.TrimSpace(out.String()), nil
}
