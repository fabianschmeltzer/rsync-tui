#!/bin/bash
# SessionStart-Hook für Claude Code im Web: stellt Go-Module und die
# CI-Werkzeuge (rsync, OpenSSH-Client, shellcheck) bereit.
set -euo pipefail

if [ "${CLAUDE_CODE_REMOTE:-}" != "true" ]; then
  exit 0
fi

cd "${CLAUDE_PROJECT_DIR:-$(dirname "$0")/../..}"

go mod download

missing=()
command -v rsync >/dev/null 2>&1 || missing+=(rsync)
command -v ssh >/dev/null 2>&1 || missing+=(openssh-client)
command -v shellcheck >/dev/null 2>&1 || missing+=(shellcheck)

if [ "${#missing[@]}" -gt 0 ]; then
  sudo_cmd=()
  [ "$(id -u)" -ne 0 ] && sudo_cmd=(sudo)
  if ! { "${sudo_cmd[@]}" apt-get update -qq &&
    DEBIAN_FRONTEND=noninteractive "${sudo_cmd[@]}" apt-get install -y -qq "${missing[@]}"; }; then
    echo "Hinweis: ${missing[*]} konnte nicht installiert werden." >&2
  fi
fi
