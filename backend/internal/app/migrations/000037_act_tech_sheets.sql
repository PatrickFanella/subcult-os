create table event_act_tech_sheets (
 id uuid primary key default gen_random_uuid(),
 event_id uuid not null,
 workspace_id uuid not null,
 act_profile_id uuid not null,
 monitor_notes text not null default '' check(char_length(monitor_notes)<=2000),
 backline_brought text not null default '' check(char_length(backline_brought)<=2000),
 backline_needed text not null default '' check(char_length(backline_needed)<=2000),
 version integer not null default 1 check(version>0),
 plot_object_key text,
 plot_content_type text check(plot_content_type in ('image/png','image/jpeg')),
 updated_by_person_id uuid not null references people(id),
 updated_at timestamptz not null default now(),
 foreign key(event_id,workspace_id) references events(id,workspace_id) on delete cascade,
 foreign key(act_profile_id,workspace_id) references cultural_profiles(id,workspace_id),
 unique(event_id,act_profile_id),unique(id,event_id,workspace_id),
 check((plot_object_key is null)=(plot_content_type is null))
);
create table event_act_inputs (
 id uuid primary key default gen_random_uuid(),
 sheet_id uuid not null,
 event_id uuid not null,
 workspace_id uuid not null,
 sort_order integer not null check(sort_order>=0),
 label text not null check(char_length(label) between 1 and 100),
 source_type text not null check(source_type in ('mic','di','line')),
 mic_di text not null default '' check(char_length(mic_di)<=100),
 stand text not null default '' check(char_length(stand)<=100),
 phantom_power boolean not null default false,
 notes text not null default '' check(char_length(notes)<=500),
 shared_key text not null default '' check(shared_key='' or shared_key ~ '^[a-z0-9][a-z0-9_-]{0,39}$'),
 foreign key(sheet_id,event_id,workspace_id) references event_act_tech_sheets(id,event_id,workspace_id) on delete cascade,
 unique(sheet_id,sort_order)
);
create unique index event_act_inputs_shared_idx on event_act_inputs(sheet_id,shared_key) where shared_key<>'';
create table event_input_list_orders (
 event_id uuid primary key,
 workspace_id uuid not null,
 revision integer not null check(revision>0),
 act_ids jsonb not null check(jsonb_typeof(act_ids)='array'),
 foreign key(event_id,workspace_id) references events(id,workspace_id) on delete cascade
);
