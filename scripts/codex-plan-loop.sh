#!/bin/bash
set -euo pipefail

repo_dir="$(pwd)"
plans_dir="docs/plan"
interval_seconds=60
once=0
dry_run=0
yolo=1
codex_bin="${CODEX_BIN:-codex}"

usage() {
  cat <<'USAGE'
Usage: scripts/codex-plan-loop.sh [options]

Find the first docs/plan/*.md document with status "Ready to implement" and run
Codex against it. When no ready plan is found, sleep and check again.

Options:
  --repo DIR         Repository root to pass to codex exec --cd (default: cwd)
  --plans-dir DIR    Plan directory relative to repo root (default: docs/plan)
  --interval SEC     Seconds to wait between checks (default: 60)
  --once             Run one scan only, then exit
  --dry-run          Print the selected plan and prompt without invoking Codex
  --yolo             Pass --yolo through to codex exec
  --codex-bin PATH   Codex executable to run (default: CODEX_BIN or codex)
  -h, --help         Show this help
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --repo)
      repo_dir="$2"
      shift 2
      ;;
    --plans-dir)
      plans_dir="$2"
      shift 2
      ;;
    --interval)
      interval_seconds="$2"
      shift 2
      ;;
    --once)
      once=1
      shift
      ;;
    --dry-run)
      dry_run=1
      shift
      ;;
    --yolo)
      yolo=1
      shift
      ;;
    --codex-bin)
      codex_bin="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if ! [[ "$interval_seconds" =~ ^[0-9]+$ ]]; then
  echo "--interval must be a non-negative integer" >&2
  exit 2
fi

if [[ ! -d "$repo_dir" ]]; then
  echo "Repository directory does not exist: $repo_dir" >&2
  exit 2
fi

absolute_plans_dir="$repo_dir/$plans_dir"

normalize_status() {
  tr '[:upper:]' '[:lower:]' \
    | sed -E 's/[*`#]//g; s/^[[:space:]]+//; s/[[:space:].]+$//'
}

plan_status() {
  local plan_file="$1"

  awk '
    BEGIN { in_status = 0 }
    {
      line = $0
      sub(/\r$/, "", line)

      if (line ~ /^##[[:space:]]*Status[[:space:]]*:?/) {
        in_status = 1
        sub(/^##[[:space:]]*Status[[:space:]]*:?[[:space:]]*/, "", line)
        if (line != "") {
          print line
          exit
        }
        next
      }

      if (line ~ /^\*\*Status:[[:space:]]*/) {
        sub(/^\*\*Status:[[:space:]]*/, "", line)
        sub(/\*\*[[:space:]]*$/, "", line)
        print line
        exit
      }

      if (in_status && line ~ /^##[[:space:]]+/) {
        exit
      }

      if (in_status && line !~ /^[[:space:]]*$/) {
        print line
        exit
      }
    }
  ' "$plan_file" | normalize_status
}

find_ready_plan() {
  local plan_file
  local status

  [[ -d "$absolute_plans_dir" ]] || return 1

  while IFS= read -r -d '' plan_file; do
    status="$(plan_status "$plan_file")"
    if [[ "$status" == "ready to implement" ]]; then
      realpath --relative-to="$repo_dir" "$plan_file"
      return 0
    fi
  done < <(find "$absolute_plans_dir" -maxdepth 1 -type f -name '*.md' -print0 | sort -z)

  return 1
}

mark_plan_ready_for_code_review() {
  local plan_path="$1"
  local plan_file="$repo_dir/$plan_path"
  local status

  status="$(plan_status "$plan_file")"
  case "$status" in
    "ready to implement"|"implementation done")
      ;;
    *)
      echo "Plan status left as \"$status\" for $plan_path"
      return 0
      ;;
  esac

  perl -0pi -e '
    my $replacement = "Ready for code review.";
    my $updated = s{(^##[ \t]*Status[ \t]*:?[ \t]*)[^\r\n]+}{${1}$replacement}mi;
    $updated ||= s{(^##[ \t]*Status[ \t]*:?[ \t]*\r?\n(?:[ \t]*\r?\n)?)[^\r\n]+}{${1}$replacement}mi;
    $updated ||= s{(^\*\*Status:\*{0,2}[ \t]*)[^*\r\n]+(\*\*)?}{${1} . $replacement . (defined($2) ? $2 : "")}mie;
  ' "$plan_file"

  echo "Updated plan status to Ready for code review: $plan_path"
}

build_prompt() {
  local plan_path="$1"

  cat <<PROMPT
Implement the technical plan at \`$plan_path\`.

Required workflow:
- Use \$incremental-implementation for thin, verifiable implementation slices.
- Use \$test-driven-development for all behavior changes.
- Read the plan first, then implement only the scope described there.
- After each slice, run the relevant tests/build checks for the files changed.
- When the plan is fully implemented and verified, update its status from "Ready to implement" to "Ready for code review".
- If the plan cannot be implemented without user input or an external dependency, update its status to "Blocked" and document the blocker in the plan.

Do not implement unrelated cleanup or neighboring feature work.
PROMPT
}

run_codex_for_plan() {
  local plan_path="$1"
  local prompt
  local -a codex_args

  prompt="$(build_prompt "$plan_path")"
  echo "Found ready plan: $plan_path"

  if [[ "$dry_run" -eq 1 ]]; then
    printf '%s\n' "$prompt"
    return 0
  fi

  codex_args=(exec --cd "$repo_dir")
  if [[ "$yolo" -eq 1 ]]; then
    codex_args+=(--yolo)
  fi
  codex_args+=(-)

  if printf '%s\n' "$prompt" | "$codex_bin" "${codex_args[@]}"; then
    mark_plan_ready_for_code_review "$plan_path"
  else
    return $?
  fi
}

while true; do
  if ready_plan="$(find_ready_plan)"; then
    if ! run_codex_for_plan "$ready_plan"; then
      echo "Codex exited non-zero for $ready_plan" >&2
    fi
  else
    echo "No ready plan found under $plans_dir"
  fi

  if [[ "$once" -eq 1 ]]; then
    exit 0
  fi

  sleep "$interval_seconds"
done
