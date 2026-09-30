-- Preserve existing Resend UUID identifiers while admitting namespaced Brevo IDs.
alter table email_outbox alter column provider_message_id type text using provider_message_id::text;
alter table email_provider_events alter column provider_message_id type text using provider_message_id::text;
-- Bind queued mail to its original provider; switching config must not resend it
-- through a provider with a different idempotency store.
alter table email_outbox add column provider text not null default 'resend'
 check (provider in ('resend','brevo'));
alter table email_outbox add constraint email_outbox_provider_id_length
 check (provider_message_id is null or length(provider_message_id) between 1 and 512);
alter table email_provider_events add constraint email_events_provider_id_length
 check (length(provider_message_id) between 1 and 512);
