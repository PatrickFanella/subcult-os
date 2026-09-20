do $$
begin
  if exists (select 1 from sessions limit 1) then
    raise exception 'prototype sessions require inventory before removal';
  end if;
end
$$;

drop table sessions;
