#!/usr/bin/env bash
# Report status of a detached `synthea-bg.sh <size>` run: pid liveness, log
# tail, bundle count on disk, and elapsed wall time. Exit 0 if running,
# 1 if finished (pidfile cleared), 2 if dead-without-cleanup (stale pid).
set -euo pipefail

size="${1:?usage: $0 <1k|10k|100k>}"
repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

synthea_dir="eval/synthea/upstream"
log="$synthea_dir/synthea-${size}.log"
pidfile="$synthea_dir/synthea-${size}.pid"
n="${size%k}000"
output_dir="$synthea_dir/output-${n}"

echo "=== synthea-${size} status ==="

if [ -f "$pidfile" ]; then
  pid="$(cat "$pidfile")"
  if kill -0 "$pid" 2>/dev/null; then
    etime="$(ps -o etime= -p "$pid" 2>/dev/null | tr -d ' ' || echo '?')"
    echo "state:    RUNNING (pid=$pid, elapsed=$etime)"
    rc=0
  else
    echo "state:    STALE PID ($pid not alive; pidfile not cleaned up)"
    rc=2
  fi
else
  echo "state:    NOT RUNNING (no pidfile)"
  rc=1
fi

if [ -d "$output_dir/fhir" ]; then
  bundles="$(ls "$output_dir/fhir" 2>/dev/null | wc -l)"
  disk="$(du -sh "$output_dir" 2>/dev/null | cut -f1)"
  echo "bundles:  $bundles on disk"
  echo "disk:     $disk in $output_dir"
else
  echo "bundles:  0 (output dir not created yet)"
fi

if [ -s "$log" ]; then
  echo "log:      $log  ($(wc -l <"$log") lines)"
  echo "--- tail ---"
  tail -n 3 "$log"
else
  echo "log:      $log  (empty or missing)"
fi

exit "$rc"
