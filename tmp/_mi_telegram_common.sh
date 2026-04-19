#!/usr/bin/env bash
set -euo pipefail

mi_telegram_repo_root() {
  cd "$(dirname "${BASH_SOURCE[0]}")/.." >/dev/null 2>&1
  pwd
}

mi_telegram_mkey_path() {
  echo "$HOME/.agents/skills/mi-key-cli/scripts/mkey.sh"
}

mi_telegram_env_path() {
  echo "$(mi_telegram_repo_root)/infra/.env"
}

mi_telegram_binary_path() {
  echo "$(mi_telegram_repo_root)/bin/mi-telegram-cli"
}

write_step() {
  printf '\n==> %s\n' "$1"
}

json_query() {
  local json_input="$1"
  local expression="$2"

  if command -v python3 >/dev/null 2>&1; then
    JSON_INPUT="$json_input" JSON_EXPR="$expression" python3 - <<'PY'
import json
import os

data = json.loads(os.environ["JSON_INPUT"])
expr = os.environ["JSON_EXPR"].split(".")
value = data
for part in expr:
    if part == "":
        continue
    if isinstance(value, dict):
        value = value.get(part)
    else:
        value = None
        break
if isinstance(value, (dict, list)):
    print(json.dumps(value))
elif value is None:
    print("")
else:
    print(value)
PY
    return
  fi

  echo "python3 is required to parse JSON in Linux smoke scripts." >&2
  exit 1
}

invoke_mi_telegram_pull() {
  local skip_pull="${1:-0}"
  if [[ "$skip_pull" == "1" ]]; then
    return
  fi

  local mkey
  mkey="$(mi_telegram_mkey_path)"
  if [[ ! -f "$mkey" ]]; then
    echo "mi-key-cli not found at $mkey" >&2
    exit 1
  fi

  write_step "Pulling secrets with mkey"
  bash "$mkey" pull mi-telegram-cli dev
}

import_mi_telegram_env() {
  local env_file
  env_file="$(mi_telegram_env_path)"
  if [[ ! -f "$env_file" ]]; then
    echo "Expected env file at $env_file. Run mkey pull first." >&2
    exit 1
  fi

  set -a
  # shellcheck disable=SC1090
  source "$env_file"
  set +a
}

ensure_mi_telegram_binary() {
  local skip_build="${1:-0}"
  local binary
  binary="$(mi_telegram_binary_path)"

  if [[ "$skip_build" == "1" && -x "$binary" ]]; then
    echo "$binary"
    return
  fi

  if [[ "$skip_build" != "1" || ! -x "$binary" ]]; then
    local repo_root
    repo_root="$(mi_telegram_repo_root)"
    mkdir -p "$(dirname "$binary")"

    write_step "Building mi-telegram-cli"
    (
      cd "$repo_root"
      go build -o "$binary" ./cmd/mi-telegram-cli
    )
  fi

  echo "$binary"
}

initialize_mi_telegram_cli() {
  local skip_pull="${1:-0}"
  local skip_build="${2:-0}"

  invoke_mi_telegram_pull "$skip_pull"
  import_mi_telegram_env
  ensure_mi_telegram_binary "$skip_build" >/dev/null
}

invoke_mi_telegram_cli_capture() {
  local allow_failure="${1:-0}"
  shift

  local binary
  binary="$(mi_telegram_binary_path)"
  if [[ ! -x "$binary" ]]; then
    echo "Binary not found at $binary. Run initialize_mi_telegram_cli first." >&2
    exit 1
  fi

  local attempt=0
  local max_retries=3
  while true; do
    set +e
    local output
    output="$("$binary" "$@" 2>&1)"
    local exit_code=$?
    set -e

    if [[ $exit_code -ne 0 && "$output" == *ProfileLocked* && $attempt -lt $max_retries ]]; then
      attempt=$((attempt + 1))
      sleep 2
      continue
    fi

    if [[ "$allow_failure" != "1" && $exit_code -ne 0 ]]; then
      echo "mi-telegram-cli failed ($exit_code): $output" >&2
      exit "$exit_code"
    fi

    printf '%s\n' "$exit_code"
    printf '%s' "$output"
    return
  done
}

ensure_mi_telegram_profile() {
  local profile="$1"
  local display_name="$2"

  local show_result
  show_result="$(invoke_mi_telegram_cli_capture 1 profiles show --profile "$profile" --json)"
  local show_exit
  show_exit="$(printf '%s\n' "$show_result" | head -n1)"
  if [[ "$show_exit" == "0" ]]; then
    return
  fi

  write_step "Creating profile $profile"
  local create_result
  create_result="$(invoke_mi_telegram_cli_capture 0 profiles add --profile "$profile" --display-name "$display_name" --json)"
  printf '%s\n' "$create_result" | tail -n +2
}

get_mi_telegram_auth_status() {
  local profile="$1"
  local result
  result="$(invoke_mi_telegram_cli_capture 1 auth status --profile "$profile" --json)"
  local exit_code
  exit_code="$(printf '%s\n' "$result" | head -n1)"
  if [[ "$exit_code" != "0" ]]; then
    echo ""
    return
  fi

  local json
  json="$(printf '%s\n' "$result" | tail -n +2)"
  json_query "$json" "data.authorizationStatus"
}

assert_mi_telegram_authorized() {
  local profile="$1"
  local status
  status="$(get_mi_telegram_auth_status "$profile")"
  if [[ "$status" != "Authorized" ]]; then
    echo "Profile '$profile' is not authorized. Run tmp/smoke-auth.sh first." >&2
    exit 1
  fi
}
