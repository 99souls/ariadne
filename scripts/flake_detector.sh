#!/usr/bin/env bash
set -euo pipefail

ITERATIONS=${ITER:-${1:-10}}
PATTERN=${PATTERN:-${2:-LiveSite}}
GO=${GO:-go}

echo "==> Flake detector: running pattern '$PATTERN' for $ITERATIONS iterations" >&2
passes=0
fails=0
durations=()

for i in $(seq 1 "$ITERATIONS"); do
  # Portable millisecond timestamp (macOS `date` lacks %N with GNU semantics)
  if start_epoch=$(date +%s 2>/dev/null); then
    if ms_part=$(date +%3N 2>/dev/null); then
      start="${start_epoch}${ms_part}"
    else
      # Fallback: use perl high‑res time * 1000 (integer)
      start=$(perl -MTime::HiRes=time -e 'printf("%d", time()*1000)')
    fi
  else
    start=$(perl -MTime::HiRes=time -e 'printf("%d", time()*1000)')
  fi
  if TESTSITE_REUSE=1 $GO test ./engine/... -run "$PATTERN" -count=1 -timeout=90s >/tmp/flake_run.$$.$i.log 2>&1; then
    if end_epoch=$(date +%s 2>/dev/null); then
      if ms_part=$(date +%3N 2>/dev/null); then
        end="${end_epoch}${ms_part}"
      else
        end=$(perl -MTime::HiRes=time -e 'printf("%d", time()*1000)')
      fi
    else
      end=$(perl -MTime::HiRes=time -e 'printf("%d", time()*1000)')
    fi
    dur=$(( end - start ))
    durations+=("$dur")
    echo "[$i] PASS ${dur}ms" >&2
    passes=$((passes+1))
  else
    echo "[$i] FAIL (see log /tmp/flake_run.$$.$i.log)" >&2
    fails=$((fails+1))
  fi
done

echo "-- Summary --" >&2
echo "Passes: $passes" >&2
echo "Fails : $fails" >&2

if [ ${#durations[@]} -gt 0 ]; then
  # basic stats
  total=0; min=999999999; max=0;
  for d in "${durations[@]}"; do
    (( d < min )) && min=$d
    (( d > max )) && max=$d
    total=$((total + d))
  done
  mean=$(( total / ${#durations[@]} ))
  # compute simple p95 (sorted, 95th percentile index)
  sorted=($(printf '%s\n' "${durations[@]}" | sort -n))
  idx=$(awk -v n=${#sorted[@]} 'BEGIN{ printf "%d", (0.95*n); }')
  (( idx >= n )) && idx=$((n-1))
  p95=${sorted[$idx]}
  echo "Durations(ms): n=${#durations[@]} min=$min max=$max mean=$mean p95=$p95" >&2
fi

if [ "$fails" -ne 0 ]; then
  echo "Detected $fails failures over $ITERATIONS iterations" >&2
  exit 1
fi

exit 0
