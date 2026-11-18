#!/bin/bash
# Start, prompt, and wait for a worker agent in a Rex session. The Rex Coordinator Card names each use.
#
#   rex_agent.sh start  <session> <agent> <cwd> -- <command>...
#   rex_agent.sh prompt <session> <agent> <timeout seconds> <one-line text>
#   rex_agent.sh wait   <session> <agent> <timeout seconds> [state]...
#   rex_agent.sh get    <session> <agent>
#
# <session> is a Rex session label. <agent> is the label of the agent's block.
# States come from the status that the agent reports to Rex: idle, working, blocked, done, error.
# "none" means that the program has reported no status yet.
# Each command prints one line. Exit codes:
#   0  start: the agent is idle. prompt: the turn ended (done or idle). wait: a listed state was reached.
#   2  the agent is blocked. For prompt before sending, the line starts with "agent_blocked" and nothing was sent.
#   3  timeout
#   4  the Rex server, the session, or the block is missing, or the arguments are wrong
#   5  agent_prompt_stalled: the text is in the input box, but the agent did not start to work
#   6  the agent reported an error
#   7  agent_busy: the agent was working, and nothing was sent
#
# The script never starts a Rex server. A server that an agent shell starts gets the file permissions of that
# agent. If no server runs, open the Rex app, then try again.
set -u
rex() { command rex --autostart=false "$@"; }
die() { echo "stop: $*"; exit 4; }
command -v jq >/dev/null || die "jq is not on PATH"
[ $# -ge 3 ] || die "usage: rex_agent.sh start|prompt|wait|get <session> <agent> ..."
ACTION="$1"; SESSION="$2"; AGENT="$3"; shift 3
rex whoami >/dev/null 2>&1 || die "no Rex server is running. Open the Rex app, then try again. This script does not start a server."

record() { rex block call -s "$SESSION" -b "$AGENT" program_status 2>/dev/null | jq -c '.records[0] // {state:"none"} | {state, kind, msg, updated_at}'; }
field() { printf '%s' "$1" | jq -r ".$2 // empty"; }
# wait_for <timeout seconds> <previous updated_at, or ""> <state>...
# Returns when the agent reports a listed state in a record that is newer than the previous one.
wait_for() {
  local limit="$1" prev="$2" end rec state; shift 2
  end=$(( $(date +%s) + limit ))
  while :; do
    rec=$(record); [ -n "$rec" ] || return 4
    state=$(field "$rec" state)
    if [ -z "$prev" ] || [ "$(field "$rec" updated_at)" != "$prev" ]; then
      for want in "$@"; do [ "$state" = "$want" ] && { REC="$rec"; return 0; }; done
    fi
    [ "$(date +%s)" -ge "$end" ] && { REC="$rec"; return 3; }
    sleep 2
  done
}
finish() {   # print the record, and exit with the code of its state
  echo "$REC"
  case "$(field "$REC" state)" in blocked) exit 2 ;; error) exit 6 ;; *) exit 0 ;; esac
}

case "$ACTION" in
  start)
    [ $# -ge 3 ] && [ "$2" = "--" ] || die "usage: rex_agent.sh start <session> <agent> <cwd> -- <command>..."
    CWD="$1"; shift 2
    [ -d "$CWD" ] || die "no folder $CWD"
    SID=$(rex ls --json | jq -r --arg l "$SESSION" '[.sessions[] | select(.label == $l)][0].session_id // empty')
    if [ -z "$SID" ]; then
      SID=$(rex new "$SESSION" -c "$HOME" --json | jq -r '.session_id // empty')
      [ -n "$SID" ] || die "cannot create the Rex session $SESSION"
    fi
    rex block ls -s "$SID" --json | jq -e --arg l "$AGENT" 'any(.. | objects; .label? == $l)' >/dev/null \
      && die "a block with the label $AGENT is already in the session $SESSION"
    rex run -s "$SID" -c "$CWD" --label "$AGENT" --focus=false --interactive --keep-open --json -- "$@" >/dev/null \
      || die "rex run failed"
    wait_for 90 "" idle blocked error done; RC=$?
    [ $RC -eq 4 ] && die "the block $AGENT closed"
    [ $RC -eq 3 ] && { echo "timeout $REC"; exit 3; }
    finish ;;
  get)
    REC=$(record); [ -n "$REC" ] || die "no block $AGENT in the session $SESSION"
    echo "$REC" ;;
  wait)
    [ $# -ge 1 ] || die "usage: rex_agent.sh wait <session> <agent> <timeout seconds> [state]..."
    LIMIT="$1"; shift
    [ $# -gt 0 ] || set -- done idle blocked error
    wait_for "$LIMIT" "" "$@"; RC=$?
    [ $RC -eq 4 ] && die "no block $AGENT in the session $SESSION"
    [ $RC -eq 3 ] && { echo "timeout $REC"; exit 3; }
    echo "$REC" ;;
  prompt)
    [ $# -eq 2 ] || die "usage: rex_agent.sh prompt <session> <agent> <timeout seconds> <one-line text>"
    LIMIT="$1"; TEXT="$2"
    case "$TEXT" in *$'\n'*) die "the text must be one line" ;; esac
    REC=$(record); [ -n "$REC" ] || die "no block $AGENT in the session $SESSION"
    case "$(field "$REC" state)" in
      blocked) echo "agent_blocked $REC"; exit 2 ;;
      working) echo "agent_busy $REC"; exit 7 ;;
    esac
    PREV=$(field "$REC" updated_at)
    rex send -s "$SESSION" -b "$AGENT" "$TEXT" || die "rex send failed"
    sleep 1
    rex send-key -s "$SESSION" -b "$AGENT" enter || die "rex send-key failed"
    wait_for 10 "$PREV" working done idle blocked error; RC=$?
    if [ $RC -eq 3 ]; then   # some agents take the first Enter as part of the paste: press Enter one more time
      rex send-key -s "$SESSION" -b "$AGENT" enter
      wait_for 10 "$PREV" working done idle blocked error; RC=$?
    fi
    [ $RC -eq 4 ] && die "the block $AGENT closed"
    [ $RC -eq 3 ] && { echo "agent_prompt_stalled $REC"; exit 5; }
    wait_for "$LIMIT" "$PREV" done idle blocked error; RC=$?
    [ $RC -eq 4 ] && die "the block $AGENT closed"
    [ $RC -eq 3 ] && { echo "timeout $REC"; exit 3; }
    finish ;;
  *) die "unknown action $ACTION" ;;
esac
