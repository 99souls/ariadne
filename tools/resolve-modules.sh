#!/usr/bin/env bash
set -euo pipefail

# Find go.mod files that are not in the repo root (root has go.work only)
# Exclude hidden directories and vendor if present.
modules=()
while IFS= read -r -d '' mod; do
  dir=$(dirname "$mod")
  modules+=("{\"workdir\":\"$dir\"}")
done < <(find . -mindepth 2 -type f -name go.mod -print0 | sort -z)

if [ ${#modules[@]} -eq 0 ]; then
  echo "No modules found" >&2
  echo "matrix={\"include\":[]}" >> "$GITHUB_OUTPUT"
  exit 0
fi

json="{\"include\":["$(IFS=,; echo "${modules[*]}")"]}"
# Fix doubled quotes introduced by above quoting; rebuild cleanly.
json="{\"include\":[${modules[*]}]}"

echo "Resolved modules matrix: $json" >&2

# GitHub Actions output
if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "matrix=$json" >> "$GITHUB_OUTPUT"
else
  echo "::set-output name=matrix::$json"  # fallback for local testing
fi
