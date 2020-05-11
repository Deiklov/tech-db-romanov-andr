CREATE OR REPLACE FUNCTION check_parent_thread() returns trigger
    language plpgsql
as
$$
DECLARE
    i           int;
    path_parent int[];
BEGIN
    select count(1)
    from (select nickname from users where lower(nickname) = lower(new.author)) nick
    into i;
    if i < 1 then
        raise exception 'not found author';
    end if;
-- проверка на thread идет в гошке

    if new.parent is not null then
        select count(1)
        from (select id from posts where thread = new.thread and id = new.parent) val
        into i;
        if i < 1 then
            raise exception 'invalid parent id';
        end if;
    else
        select count(id) from posts where thread = new.thread and cardinality(path) = 2 into i;
        new.path = ARRAY [0] || i;
    end if;

    if new.parent is not null then
        select path from posts where id = new.parent into path_parent;
        select count(id)
        from posts
        where thread = new.thread
          and cardinality(path_parent) + 1 = cardinality(path)
          and parent = new.parent
        into i;

        new.path = path_parent || i;
    end if;

    RETURN NEW;
END;
$$;

alter function check_parent_thread() owner to docker;

create or replace function get_nickname() returns trigger
    language plpgsql
as
$$
DECLARE
    nick varchar;
BEGIN

    select nickname from users where lower(nickname) = lower(new.nickname) into nick;
    new.nickname := nick;
    RETURN NEW;
END;
$$;

alter function get_nickname() owner to docker;

create or replace function handler_data() returns trigger
    language plpgsql
as
$$
DECLARE
    voice int2;
BEGIN
    if new.votes then
        voice = 1;
    else
        voice = -1;
    end if;

    if (TG_OP = 'UPDATE') THEN
        if old.votes = new.votes then
        else
            if new.votes then
                voice = 2;
            else
                voice = -2;
            end if;
            update threads set votes=votes + (voice) where id = new.thread_id;
        end if;
        return new;
    end if;

    update threads set votes=votes + (voice) where id = new.thread_id;
    RETURN NEW;
END;
$$;

alter function handler_data() owner to docker;

create function inc_params() returns trigger
    language plpgsql
as
$$
declare
    forum_slug text;
begin
    forum_slug = new.forum;
    if tg_name = 'inc_threads' then
        update forums set threads=threads + 1 where lower(slug) = lower(forum_slug);
    elsif tg_name = 'inc_posts' then
        update forums set posts=posts + 1 where lower(slug) = lower(forum_slug);
    end if;
    return new;
end;
$$;

alter function inc_params() owner to docker;

