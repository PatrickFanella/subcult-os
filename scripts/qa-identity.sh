#!/usr/bin/env bash
# Sourced by local rehearsal scripts. Never use this mailbox reader on retained data.

qa_require_disposable_target() {
  command -v node psql curl >/dev/null || return 1
  api_url="${api_url%/}"
  API_URL="${api_url:-}" QA_DATABASE_URL="${QA_DATABASE_URL:-}" QA_DISPOSABLE_DATABASE="${QA_DISPOSABLE_DATABASE:-}" node <<'JS'
const loopback = new Set(['localhost', '127.0.0.1', '[::1]']);
try {
  const api = new URL(process.env.API_URL);
  const db = new URL(process.env.QA_DATABASE_URL);
  if (process.env.QA_DISPOSABLE_DATABASE !== '1'
    || api.protocol !== 'http:' || !loopback.has(api.hostname)
    || api.username || api.password || api.search || api.hash || api.pathname !== '/'
    || !['postgres:', 'postgresql:'].includes(db.protocol) || !loopback.has(db.hostname)
    || !/^\/subcult_qa_[a-z0-9_]+$/.test(db.pathname) || db.hash
    || [...db.searchParams.keys()].some(key => key !== 'sslmode')
    || (db.searchParams.has('sslmode') && !['disable', 'prefer', 'require'].includes(db.searchParams.get('sslmode')))) throw new Error();
} catch {
  console.error('Rehearsal requires loopback API_URL, loopback QA_DATABASE_URL named subcult_qa_*, and QA_DISPOSABLE_DATABASE=1.');
  process.exit(1);
}
JS
}

qa_verify_signup() {
  local cookie="$1" recipient="$2" body payload
  qa_require_disposable_target || return 1
  if [[ ! "$recipient" =~ ^[a-zA-Z0-9+._-]+@example\.test$ ]]; then
    echo 'Rehearsal verification only accepts synthetic example.test recipients.' >&2
    return 1
  fi
  # Read only this synthetic recipient's held, unconsumed verification message.
  body="$(psql "$QA_DATABASE_URL" -XAt --set=ON_ERROR_STOP=1 --set=recipient="$recipient" <<'SQL'
select o.body from email_outbox o
join identity_challenges c on c.id = o.related_id
where o.recipient_email = :'recipient' and o.related_type = 'identity_verification'
  and o.delivery_status = 'held' and c.consumed_at is null and c.expires_at > now()
order by o.created_at desc limit 1;
SQL
  )" || return 1
  payload="$(printf '%s' "$body" | node -e '
let body=""; process.stdin.on("data", chunk => body += chunk); process.stdin.on("end", () => {
  try {
    const links=body.split(/\s+/).filter(part=>part.startsWith("http://"));
    if(links.length!==1) throw new Error();
    const url=new URL(links[0]); const token=url.searchParams.get("token");
    if(!["localhost","127.0.0.1","[::1]"].includes(url.hostname) || url.pathname!=="/verify-email" || !token) throw new Error();
    process.stdout.write(JSON.stringify({token}));
  } catch { console.error("No valid local verification message for synthetic account."); process.exitCode=1; }
});')" || return 1
  # The normal endpoint consumes the challenge and issues the session; no DB writes.
  printf '%s' "$payload" | curl -fsS -c "$cookie" -H 'Content-Type: application/json' \
    --data-binary @- "${api_url}/api/auth/verify-email" >/dev/null
}

# Print the newest unconsumed verify or recover link for a synthetic recipient.
# Device rehearsals open it on a phone; the link itself is not consumed here.
qa_identity_link() {
  local recipient="$1" purpose="$2" related_type path body
  qa_require_disposable_target || return 1
  case "$purpose" in
    verify) related_type=identity_verification path=/verify-email ;;
    recover) related_type=identity_recovery path=/recover-password ;;
    *) echo 'Link purpose must be verify or recover.' >&2; return 1 ;;
  esac
  if [[ ! "$recipient" =~ ^[a-zA-Z0-9+._-]+@example\.test$ ]]; then
    echo 'Rehearsal links are only read for synthetic example.test recipients.' >&2
    return 1
  fi
  body="$(psql "$QA_DATABASE_URL" -XAt --set=ON_ERROR_STOP=1 --set=recipient="$recipient" --set=related_type="$related_type" <<'SQL'
select o.body from email_outbox o
join identity_challenges c on c.id = o.related_id
where o.recipient_email = :'recipient' and o.related_type = :'related_type'
  and o.delivery_status = 'held' and c.consumed_at is null and c.expires_at > now()
order by o.created_at desc limit 1;
SQL
  )" || return 1
  printf '%s' "$body" | LINK_PATH="$path" node -e '
let body=""; process.stdin.on("data", chunk => body += chunk); process.stdin.on("end", () => {
  try {
    const links=body.split(/\s+/).filter(part=>/^https?:\/\//.test(part));
    if(links.length!==1) throw new Error();
    const url=new URL(links[0]);
    if(url.pathname!==process.env.LINK_PATH || !url.searchParams.get("token")) throw new Error();
    process.stdout.write(url.href+"\n");
  } catch { console.error("No valid unconsumed identity message for synthetic account."); process.exitCode=1; }
});'
}
