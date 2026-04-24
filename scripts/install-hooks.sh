#!/usr/bin/env bash
# Wire .githooks/ into this clone. Idempotent.
set -euo pipefail
repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"
git config core.hooksPath .githooks
echo "✓ core.hooksPath = .githooks"
ls -l .githooks/
