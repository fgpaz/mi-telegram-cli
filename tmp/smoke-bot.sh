#!/usr/bin/env bash
set -euo pipefail

# Git Bash / MSYS can rewrite slash-leading text payloads like "/start <token>"
# before they reach the CLI. Keep direct smoke sends literal.
export MSYS_NO_PATHCONV=1

PROFILE="qa-dev"
PEER=""
TIMEOUT_SEC=60
READ_LIMIT=5
TEXT=""
SKIP_PULL=0
SKIP_BUILD=0
SKIP_AUTH_CHECK=0
MARK_READ=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --profile)
      PROFILE="$2"
      shift 2
      ;;
    --peer)
      PEER="$2"
      shift 2
      ;;
    --timeout-sec)
      TIMEOUT_SEC="$2"
      shift 2
      ;;
    --read-limit)
      READ_LIMIT="$2"
      shift 2
      ;;
    --text)
      TEXT="$2"
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
    --skip-auth-check)
      SKIP_AUTH_CHECK=1
      shift
      ;;
    --mark-read)
      MARK_READ=1
      shift
      ;;
    *)
      echo "Unknown arg: $1" >&2
      exit 1
      ;;
  esac
done

if [[ -z "$PEER" ]]; then
  echo "--peer is required" >&2
  exit 1
fi

if (( TIMEOUT_SEC < 1 || TIMEOUT_SEC > 300 )); then
  echo "--timeout-sec must be between 1 and 300" >&2
  exit 1
fi

# shellcheck disable=SC1091
source "$(dirname "$0")/_mi_telegram_common.sh"

initialize_mi_telegram_cli "$SKIP_PULL" "$SKIP_BUILD"

if [[ "$SKIP_AUTH_CHECK" != "1" ]]; then
  assert_mi_telegram_authorized "$PROFILE"
fi

if [[ -z "$TEXT" ]]; then
  if command -v python3 >/dev/null 2>&1; then
    TEXT="smoke-$(python3 - <<'PY'
import uuid
print(uuid.uuid4().hex[:10])
PY
)"
  else
    TEXT="smoke-$(date +%s)"
  fi
fi

write_step "messages send"
SEND_OUTPUT="$(invoke_mi_telegram_cli_capture 0 messages send --profile "$PROFILE" --peer "$PEER" --text "$TEXT" --json)"
SEND_JSON="$(printf '%s\n' "$SEND_OUTPUT" | tail -n +2)"
printf '%s\n' "$SEND_JSON"
MESSAGE_ID="$(json_query "$SEND_JSON" "data.messageId")"

write_step "messages wait"
WAIT_OUTPUT="$(invoke_mi_telegram_cli_capture 0 messages wait --profile "$PROFILE" --peer "$PEER" --after-id "$MESSAGE_ID" --timeout "$TIMEOUT_SEC" --json)"
printf '%s\n' "$WAIT_OUTPUT" | tail -n +2

write_step "messages read"
READ_OUTPUT="$(invoke_mi_telegram_cli_capture 0 messages read --profile "$PROFILE" --peer "$PEER" --limit "$READ_LIMIT" --after-id "$MESSAGE_ID" --json)"
printf '%s\n' "$READ_OUTPUT" | tail -n +2

if [[ "$MARK_READ" == "1" ]]; then
  write_step "dialogs mark-read"
  MARK_READ_OUTPUT="$(invoke_mi_telegram_cli_capture 0 dialogs mark-read --profile "$PROFILE" --peer "$PEER" --json)"
  printf '%s\n' "$MARK_READ_OUTPUT" | tail -n +2
fi

write_step "summary"
printf '{\n  "profile": "%s",\n  "peer": "%s",\n  "sentText": "%s",\n  "messageId": %s\n}\n' \
  "$PROFILE" "$PEER" "$TEXT" "$MESSAGE_ID"
