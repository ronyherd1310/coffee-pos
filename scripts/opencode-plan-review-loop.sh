#!/bin/bash
set -euo pipefail

repo_dir="$(pwd)"
plans_dir="docs/plan"
reviews_dir="docs/reviews"
interval_seconds=60
once=0
dry_run=0
skip_permissions=1
opencode_bin="${OPENCODE_BIN:-opencode}"

usage() {
  cat <<'USAGE'
Usage: scripts/opencode-plan-review-loop.sh [options]

Find the first docs/plan/*.md document with status "Ready for code review" that
does not already have a review file, then run OpenCode to review it and write
findings under docs/reviews. When no reviewable plan is found, sleep and check
again.

Options:
  --repo DIR                         Repository root to pass to opencode run --dir (default: cwd)
  --plans-dir DIR                    Plan directory relative to repo root (default: docs/plan)
  --reviews-dir DIR                  Review directory relative to repo root (default: docs/reviews)
  --interval SEC                     Seconds to wait between checks (default: 60)
  --once                             Run one scan only, then exit
  --dry-run                          Print the selected plan/review paths and prompt without invoking OpenCode
  --dangerously-skip-permissions     Pass --dangerously-skip-permissions to opencode run (default)
  --no-dangerously-skip-permissions  Do not pass --dangerously-skip-permissions
  --opencode-bin PATH                OpenCode executable to run (default: OPENCODE_BIN or opencode)
  -h, --help                         Show this help
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
    --dangerously-skip-permissions)
      skip_permissions=1
      shift
      ;;
    --no-dangerously-skip-permissions)
      skip_permissions=0
      shift
      ;;
    --opencode-bin)
      opencode_bin="$2"
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
absolute_reviews_dir="$repo_dir/$reviews_dir"

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

review_status() {
  local review_file="$1"

  [[ -f "$review_file" ]] || return 0

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

review_stem_for_plan() {
  local plan_path="$1"
  local file_name
  local stem

  file_name="$(basename "$plan_path")"
  stem="${file_name%.md}"
  stem="${stem%-tech-plan}"
  stem="${stem%-plan}"
  printf '%s\n' "$stem"
}

review_path_for_plan() {
  local plan_path="$1"
  local stem

  stem="$(review_stem_for_plan "$plan_path")"
  printf '%s/%s-review.md\n' "$reviews_dir" "$stem"
}

has_review_for_plan() {
  local plan_path="$1"
  local stem

  stem="$(review_stem_for_plan "$plan_path")"
  [[ -d "$absolute_reviews_dir" ]] || return 1

  find "$absolute_reviews_dir" -maxdepth 1 -type f -name "$stem-review*.md" -print -quit \
    | grep -q .
}

find_reviewable_plan() {
  local plan_file
  local status

  [[ -d "$absolute_plans_dir" ]] || return 1

  while IFS= read -r -d '' plan_file; do
    status="$(plan_status "$plan_file")"
    if [[ "$status" == "ready for code review" ]] && ! has_review_for_plan "$plan_file"; then
      realpath --relative-to="$repo_dir" "$plan_file"
      return 0
    fi
  done < <(find "$absolute_plans_dir" -maxdepth 1 -type f -name '*.md' -print0 | sort -z)

  return 1
}

mark_review_ready_to_be_checked() {
  local review_path="$1"
  local review_file="$repo_dir/$review_path"
  local status

  if [[ ! -f "$review_file" ]]; then
    echo "Review file was not created: $review_path" >&2
    return 1
  fi

  status="$(review_status "$review_file")"
  if [[ "$status" == "blocked" ]]; then
    echo "Review status left as \"$status\" for $review_path"
    return 0
  fi

  perl -0pi -e '
    my $replacement = "Ready to be checked";
    my $updated = s{(^##[ \t]*Status[ \t]*:?[ \t]*)[^\r\n]+}{${1}$replacement}mi;
    $updated ||= s{(^##[ \t]*Status[ \t]*:?[ \t]*\r?\n(?:[ \t]*\r?\n)?)[^\r\n]+}{${1}$replacement}mi;
    $updated ||= s{(^\*\*Status:\*{0,2}[ \t]*)[^*\r\n]+(\*\*)?}{${1} . $replacement . (defined($2) ? $2 : "")}mie;
    if (!$updated) {
      $updated = s{(^#[^\r\n]*\r?\n)}{$1\n**Status:** $replacement\n}m;
    }
    if (!$updated) {
      $_ = "**Status:** $replacement\n\n$_";
    }
  ' "$review_file"

  echo "Updated review status to Ready to be checked: $review_path"
}

build_prompt() {
  local plan_path="$1"
  local review_path="$2"

  cat <<PROMPT
Code review the implementation for the technical plan at \`$plan_path\`.

Required workflow:
- Use \$code-review-and-quality before reviewing.
- Read the plan, repository guidance, referenced specs/contracts, and the relevant implementation files.
- Review the actual code that implements the plan; do not only summarize the plan.
- Create the review findings file at \`$review_path\`.
- If the review finds issues, lead with findings ordered by severity and include concrete file/line references.
- If there are no findings, write that clearly and mention remaining test gaps or residual risk.
- Include the verification commands you ran, or explicitly state why verification could not be run.
- When the review is complete, mark \`$review_path\` with status "Ready to be checked".
- Do not modify implementation code as part of this review loop. Only create or update the review file.
- If the review is blocked, still create \`$review_path\` with the blocker and the context needed to continue.

Use the existing review documents in \`$reviews_dir\` as the formatting baseline.
PROMPT
}

run_opencode_for_plan() {
  local plan_path="$1"
  local review_path
  local prompt
  local -a opencode_args

  mkdir -p "$absolute_reviews_dir"

  review_path="$(review_path_for_plan "$plan_path")"
  prompt="$(build_prompt "$plan_path" "$review_path")"
  echo "Found ready-for-code-review plan: $plan_path"
  echo "Expected review file: $review_path"

  if [[ "$dry_run" -eq 1 ]]; then
    printf '%s\n' "$prompt"
    return 0
  fi

  opencode_args=(run --dir "$repo_dir" --title "Review $plan_path")
  if [[ "$skip_permissions" -eq 1 ]]; then
    opencode_args+=(--dangerously-skip-permissions)
  fi
  opencode_args+=("$prompt")

  if "$opencode_bin" "${opencode_args[@]}"; then
    mark_review_ready_to_be_checked "$review_path"
  else
    return $?
  fi
}

while true; do
  if reviewable_plan="$(find_reviewable_plan)"; then
    if ! run_opencode_for_plan "$reviewable_plan"; then
      echo "OpenCode exited non-zero for $reviewable_plan" >&2
    fi
  else
    echo "No ready-for-code-review plan without review found under $plans_dir"
  fi

  if [[ "$once" -eq 1 ]]; then
    exit 0
  fi

  sleep "$interval_seconds"
done
