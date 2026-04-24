#!/usr/bin/env bash
# Detached background launcher for `make synthea-<size>`.
#
# `Bash(run_in_background=true)` from Claude Code leaves the process in the
# Claude session's process group — when Claude exits, SIGHUP cascades and
# the job dies. This launcher uses `setsid --fork` so the child becomes its
# own session leader and survives parent exit. A pidfile lets callers check
# liveness reliably; a double-launch guard refuses to start if one is alive.
#
# Usage:  scripts/synthea-bg.sh <1k|10k|100k>
# See:    scripts/synthea-bg-status.sh, `make synthea-100k-bg`
set -euo pipefail

size="${1:?usage: $0 <1k|10k|100k>}"
case "$size" in 1k|10k|100k) ;; *) echo "bad size: $size" >&2; exit 2;; esac

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

synthea_dir="eval/synthea/upstream"
log="$synthea_dir/synthea-${size}.log"
pidfile="$synthea_dir/synthea-${size}.pid"

if [ -f "$pidfile" ] && kill -0 "$(cat "$pidfile")" 2>/dev/null; then
  echo "✗ synthea-${size} already running (pid $(cat "$pidfile")); tail $log" >&2
  exit 1
fi
rm -f "$pidfile"

# Archive any prior log so we don't confuse status tooling.
if [ -s "$log" ]; then
  mv "$log" "${log%.log}-$(date -u +%Y%m%dT%H%M%SZ).log"
fi

# `setsid --fork` forks a child that becomes a new session leader. The child
# records its own PID, runs the make target, then clears the pidfile on exit.
# stdin </dev/null, stdout+stderr > log keep the detached child from holding
# any of our file descriptors.
setsid --fork bash <<EOS >/dev/null 2>&1 </dev/null
  echo \$\$ > "$pidfile"
  {
    echo "start: \$(date -u +%FT%TZ) pid=\$\$ size=${size}"
    make synthea-${size}
    rc=\$?
    echo "end: \$(date -u +%FT%TZ) rc=\$rc"
    exit \$rc
  } > "$log" 2>&1
  rm -f "$pidfile"
EOS

# Wait briefly for pidfile to appear, then confirm.
for _ in 1 2 3 4 5 6 7 8 9 10; do
  [ -s "$pidfile" ] && break
  sleep 0.2
done
if [ ! -s "$pidfile" ]; then
  echo "✗ pidfile not written — launch failed? see $log" >&2
  exit 1
fi
pid="$(cat "$pidfile")"
if ! kill -0 "$pid" 2>/dev/null; then
  echo "✗ pid=$pid not alive immediately after launch; see $log" >&2
  exit 1
fi
echo "✓ synthea-${size} launched detached:"
echo "    pid      $pid"
echo "    log      $log"
echo "    pidfile  $pidfile"
echo "  status:  scripts/synthea-bg-status.sh $size"
echo "  stop:    kill \$(cat $pidfile)"
