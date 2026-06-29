#!/bin/bash
set -euo pipefail

repo_dir="$(pwd)"
reviews_dir="docs/reviews"
interval_seconds=60
once=0
dry_run=0
yolo=1
codex_bin="${CODEX_BIN:-codex}"

usage() {
  cat <<'USAGE'
Usage: scripts/codex-review-fix-loop.sh [options]

Find the first docs/reviews/*.md document with status "Ready to be checked" and
run Codex to fix all critical findings in that review. When no ready review is
found, sleep and check again.

Options:
  --repo DIR         Repository root to pass to codex exec --cd (default: cwd)
  --reviews-dir DIR  Review directory relative to repo root (default: docs/reviews)
  --interval SEC     Seconds to wait between checks (default: 60)
  --once             Run one scan only, then exit
  --dry-run          Print the selected review and prompt without invoking Codex
  --yolo             Pass --yolo through to codex exec (default)
  --no-yolo          Do not pass --yolo to codex exec
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
    --reviews-dir)
      reviews_dir="$2"
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
    --no-yolo)
      yolo=0
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

absolute_reviews_dir="$repo_dir/$reviews_dir"

normalize_status() {
  tr '[:upper:]' '[:lower:]' \
    | sed -E 's/[*`#]//g; s/^[[:space:]]+//; s/[[:space:].]+$//'
}

review_status() {
  local review_file="$1"

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
  ' "$review_file" | normalize_status
}

find_ready_review() {
  local review_file
  local status

  [[ -d "$absolute_reviews_dir" ]] || return 1

  while IFS= read -r -d '' review_file; do
    status="$(review_status "$review_file")"
    if [[ "$status" == "ready to be checked" ]]; then
      realpath --relative-to="$repo_dir" "$review_file"
      return 0
    fi
  done < <(find "$absolute_reviews_dir" -maxdepth 1 -type f -name '*.md' -print0 | sort -z)

  return 1
}

mark_review_ready_to_be_merged() {
  local review_path="$1"
  local review_file="$repo_dir/$review_path"
  local status

  status="$(review_status "$review_file")"
  case "$status" in
    "ready to be checked"|"done fixing")
      ;;
    *)
      echo "Review status left as \"$status\" for $review_path"
      return 0
      ;;
  esac

  perl -0pi -e '
    my $replacement = "Ready to be merged";
    my $updated = s{(^##[ \t]*Status[ \t]*:?[ \t]*)[^\r\n]+}{${1}$replacement}mi;
    $updated ||= s{(^##[ \t]*Status[ \t]*:?[ \t]*\r?\n(?:[ \t]*\r?\n)?)[^\r\n]+}{${1}$replacement}mi;
    $updated ||= s{(^\*\*Status:\*{0,2}[ \t]*)[^*\r\n]+(\*\*)?}{${1} . $replacement . (defined($2) ? $2 : "")}mie;
  ' "$review_file"

  echo "Updated review status to Ready to be merged: $review_path"
}

build_prompt() {
  local review_path="$1"

  cat <<PROMPT
Fix the critical code review findings in \`$review_path\`.

Required workflow:
- Use \$debugging-and-error-recovery to understand and reproduce each critical finding.
- Use \$test-driven-development for behavior changes; add or update focused tests before fixing when practical.
- Read the review document first, then inspect only the relevant plan/spec/source/test files needed for the critical findings.
- Fix all findings explicitly marked Critical. Do not fix High, Medium, Low, Nit, or follow-up items unless they are required to resolve a Critical finding.
- After implementation, run the relevant verification commands for the changed files.
- Update \`$review_path\` when finished:
  - Change status from "Ready to be checked" to "Ready to be merged" if every Critical finding is fixed and verified.
  - Change status to "Blocked" if a Critical finding cannot be fixed without user input or an external dependency, and document the blocker.
  - If the review contains no Critical findings, change status to "No Critical Findings" and document that no code changes were needed.
- Include a concise fix summary and verification results in the review document.

Do not implement unrelated cleanup or neighboring feature work.
PROMPT
}

run_codex_for_review() {
  local review_path="$1"
  local prompt
  local -a codex_args

  prompt="$(build_prompt "$review_path")"
  echo "Found ready review: $review_path"

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
    mark_review_ready_to_be_merged "$review_path"
  else
    return $?
  fi
}

while true; do
  if ready_review="$(find_ready_review)"; then
    if ! run_codex_for_review "$ready_review"; then
      echo "Codex exited non-zero for $ready_review" >&2
    fi
  else
    echo "No ready review found under $reviews_dir"
  fi

  if [[ "$once" -eq 1 ]]; then
    exit 0
  fi

  sleep "$interval_seconds"
done
