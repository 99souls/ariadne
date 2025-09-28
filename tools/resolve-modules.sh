#!/usr/bin/env bash
set -euo pipefail

# Discover module directories (exclude root which has go.work only)
modules=()
while IFS= read -r -d '' mod; do
  dir=$(dirname "$mod")
  dir=${dir#./}              # normalize: remove leading ./
  modules+=("{\"workdir\":\"$dir\"}")
done < <(find . -mindepth 2 -type f -name go.mod -print0 | sort -z)

if [ ${#modules[@]} -eq 0 ]; then
  echo "No modules found" >&2
  if [ -n "${GITHUB_OUTPUT:-}" ]; then
    echo "matrix={\"include\":[]}" >> "$GITHUB_OUTPUT"
  else
    echo "::set-output name=matrix::{\"include\":[]}"  # legacy fallback
  fi
  exit 0
fi

# Build valid JSON array with commas
json='{"include":['
for i in "${!modules[@]}"; do
  if [ "$i" -ne 0 ]; then json+=','; fi
  json+="${modules[$i]}"
done
json+=']}'

echo "Resolved modules matrix: $json" >&2

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "matrix=$json" >> "$GITHUB_OUTPUT"
else
  echo "::set-output name=matrix::$json"  # legacy fallback for local runs
fi
