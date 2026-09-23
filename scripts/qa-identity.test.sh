#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=scripts/qa-identity.sh
source "$(dirname -- "${BASH_SOURCE[0]}")/qa-identity.sh"

# No network or database access. The source helper must propagate every failure.
psql() { printf '%s' "${test_mail_body:-Verify account http://127.0.0.1:5198/verify-email?token=synthetic-challenge}"; }
curl() {
  node -e 'let input="";process.stdin.on("data",c=>input+=c);process.stdin.on("end",()=>{if(JSON.parse(input).token!=="synthetic-challenge")process.exitCode=1;});' || return 1
  return "${test_http_status:-0}"
}

api_url=http://127.0.0.1:8098
QA_DATABASE_URL=postgres://test:synthetic@127.0.0.1:5432/subcult_qa_helper
QA_DISPOSABLE_DATABASE=1
qa_require_disposable_target
qa_verify_signup /tmp/unused-synthetic.cookies synthetic@example.test

expect_failure() {
  if "$@" >/dev/null 2>&1; then
    echo "Expected rejection: $1" >&2
    exit 1
  fi
}
QA_DISPOSABLE_DATABASE=0 expect_failure qa_require_disposable_target
api_url=https://subcults.subcult.tv expect_failure qa_require_disposable_target
api_url=http://127.0.0.1:8098/private expect_failure qa_require_disposable_target
QA_DATABASE_URL=postgres://test@remote.example/subcult_qa_helper expect_failure qa_require_disposable_target
QA_DATABASE_URL=postgres://test@localhost/subcult_production expect_failure qa_require_disposable_target
QA_DATABASE_URL='postgres://test@localhost/subcult_qa_helper?host=remote.example' expect_failure qa_require_disposable_target
expect_failure qa_verify_signup /tmp/unused-synthetic.cookies real@example.com
test_mail_body='No message' expect_failure qa_verify_signup /tmp/unused-synthetic.cookies synthetic@example.test
test_mail_body='https://remote.example/verify-email?token=synthetic-challenge' expect_failure qa_verify_signup /tmp/unused-synthetic.cookies synthetic@example.test
test_http_status=22 expect_failure qa_verify_signup /tmp/unused-synthetic.cookies synthetic@example.test
psql() { return 9; }
expect_failure qa_verify_signup /tmp/unused-synthetic.cookies synthetic@example.test
echo 'QA identity guards and verification failure propagation passed.'
