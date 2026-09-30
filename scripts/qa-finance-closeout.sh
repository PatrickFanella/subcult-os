#!/usr/bin/env bash
# Fresh-record, free-only finance/closeout/reuse API rehearsal. No paid checkout.
set -euo pipefail
umask 077

api_url="${API_URL:-}"
# shellcheck source=scripts/qa-identity.sh
source "$(dirname -- "${BASH_SOURCE[0]}")/qa-identity.sh"
qa_require_disposable_target
command -v python3 >/dev/null

tmpdir="$(mktemp -d /tmp/subcult-finance-qa.XXXXXX)"
trap 'rm -rf "${tmpdir}"' EXIT
suffix="$(date +%s%N)"
owner="finance-owner+${suffix}@example.test"
member="finance-crew+${suffix}@example.test"

signup() {
  local email="$1" cookie="$2"
  curl -fsS -c "$cookie" -H 'Content-Type: application/json' \
    -d "{\"email\":\"${email}\",\"password\":\"secret1234\",\"displayName\":\"Synthetic finance rehearsal\"}" \
    "${api_url}/api/auth/signup" >/dev/null
  qa_verify_signup "$cookie" "$email"
}
signup "$owner" "$tmpdir/owner.cookies"
signup "$member" "$tmpdir/member.cookies"

python3 "$(dirname -- "${BASH_SOURCE[0]}")/qa-finance-closeout.py" \
  "$api_url" "$tmpdir/owner.cookies" "$tmpdir/member.cookies" "$member"
