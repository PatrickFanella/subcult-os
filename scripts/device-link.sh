#!/usr/bin/env bash
# Print the newest verification or recovery link for a synthetic account in a
# disposable rehearsal stack, and optionally open it on a USB-connected Android
# device. See docs/development/mobile-app-links.md.
set -euo pipefail
# shellcheck source=scripts/qa-identity.sh
source "$(dirname -- "${BASH_SOURCE[0]}")/qa-identity.sh"

if [[ $# -lt 2 || $# -gt 3 || ( $# -eq 3 && "$3" != --adb ) ]]; then
  echo 'Usage: API_URL=http://127.0.0.1:PORT QA_DATABASE_URL=... QA_DISPOSABLE_DATABASE=1 scripts/device-link.sh verify|recover RECIPIENT@example.test [--adb]' >&2
  exit 2
fi
api_url="${API_URL:-}"
link="$(qa_identity_link "$2" "$1")"
printf '%s\n' "$link"
if [[ "${3:-}" == --adb ]]; then
  adb shell am start -W -a android.intent.action.VIEW -c android.intent.category.BROWSABLE -d "'$link'"
fi
