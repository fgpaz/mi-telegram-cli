#!/usr/bin/env bash
set -euo pipefail

PROFILE="qa-dev"
DISPLAY_NAME="QA Dev"
DIALOGS_LIMIT=10
SKIP_PULL=0
SKIP_BUILD=0
SKIP_DIALOGS=0
FORCE_LOGIN=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --profile)
      PROFILE="$2"
      shift 2
      ;;
    --display-name)
      DISPLAY_NAME="$2"
      shift 2
      ;;
    --dialogs-limit)
      DIALOGS_LIMIT="$2"
      shift 2
      ;;
    --skip-pull)
      SKIP_PULL=1
      shift
      ;;
    --skip-build)
      SKIP_BUILD=1
      shift
      ;;
    --skip-dialogs)
      SKIP_DIALOGS=1
      shift
      ;;
    --force-login)
      FORCE_LOGIN=1
      shift
      ;;
    *)
      echo "Unknown arg: $1" >&2
      exit 1
      ;;
  esac
done

# shellcheck disable=SC1091
source "$(dirname "$0")/_mi_telegram_common.sh"

initialize_mi_telegram_cli "$SKIP_PULL" "$SKIP_BUILD"
ensure_mi_telegram_profile "$PROFILE" "$DISPLAY_NAME"

AUTH_STATUS="$(get_mi_telegram_auth_status "$PROFILE")"
if [[ "$FORCE_LOGIN" == "1" || "$AUTH_STATUS" != "Authorized" ]]; then
  login_succeeded=0
  for attempt in 1 2 3 4 5; do
    write_step "QR login for profile $PROFILE (attempt $attempt/5)"
    login_log="$(mktemp)"
    set +e
    "$(mi_telegram_binary_path)" auth login --profile "$PROFILE" --method qr 2>&1 | tee "$login_log"
    login_exit=${PIPESTATUS[0]}
    set -e
    login_output="$(cat "$login_log")"
    rm -f "$login_log"

    if [[ $login_exit -eq 0 ]]; then
      login_succeeded=1
      break
    fi

    if [[ "$login_output" == *ProfileLocked* && "$attempt" != "5" ]]; then
      echo "Profile lock is busy; waiting before retry..." >&2
      sleep 2
      continue
    fi

    echo "QR login failed with exit code $login_exit" >&2
    exit "$login_exit"
  done

  if [[ $login_succeeded -ne 1 ]]; then
    echo "QR login could not acquire the profile lock after multiple attempts." >&2
    exit 1
  fi
else
  write_step "Profile $PROFILE is already authorized"
fi

write_step "auth status"
STATUS_OUTPUT="$(invoke_mi_telegram_cli_capture 0 auth status --profile "$PROFILE" --json)"
printf '%s\n' "$STATUS_OUTPUT" | tail -n +2

write_step "me"
ME_OUTPUT="$(invoke_mi_telegram_cli_capture 0 me --profile "$PROFILE" --json)"
printf '%s\n' "$ME_OUTPUT" | tail -n +2

if [[ "$SKIP_DIALOGS" != "1" ]]; then
  write_step "dialogs list"
  DIALOGS_OUTPUT="$(invoke_mi_telegram_cli_capture 0 dialogs list --profile "$PROFILE" --limit "$DIALOGS_LIMIT" --json)"
  printf '%s\n' "$DIALOGS_OUTPUT" | tail -n +2
fi
