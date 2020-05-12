create table if not exists users
(
    nickname varchar(128) not null
        constraint users_pk
            primary key,
    fullname varchar(128) not null,
    about text,
    email varchar(128) not null
);

alter table users owner to docker;

create unique index if not exists users_lower_idx
    on users (lower(nickname::text));

create unique index if not exists users_lower_idx1
    on users (lower(email::text));

create unique index if not exists users_nickname_idx
    on users (nickname);

create unique index if not exists users_nickname_fullname_about_email_idx
    on users (nickname, fullname, about, email);

create table if not exists forums
(
    posts integer default 0 not null,
    slug varchar(128) not null
        constraint forums_pk
            primary key,
    threads integer default 0 not null,
    title varchar(256) not null,
    "user" varchar(128) not null
        constraint forums_users_nickname_fk
            references users
            on update set null on delete set null
);

alter table forums owner to docker;

create unique index if not exists forums_lower_idx
    on forums (lower(slug::text));

create index if not exists forums_user_idx
    on forums ("user");

create table if not exists threads
(
    author varchar(128) not null
        constraint threads_users_nickname_fk
            references users,
    created timestamp not null,
    forum varchar(128) not null
        constraint threads_forums_slug_fk
            references forums
            on update cascade on delete cascade,
    id serial not null
        constraint threads_pk
            primary key,
    message text not null,
    slug varchar(128),
    title varchar(256) not null,
    votes integer default 0 not null
);

alter table threads owner to docker;

create unique index if not exists threads_lower_idx
    on threads (lower(slug::text));

create unique index if not exists threads_slug_uindex
    on threads (slug);

create index if not exists threads_forum_index
    on threads (forum);

create index if not exists threads_author_index
    on threads (author);

create unique index if not exists threads_id_votes_idx
    on threads (id, votes);

create index if not exists threads_lower_created_idx
    on threads (lower(forum::text), created);

create index if not exists threads_lower_created_idx1
    on threads (lower(forum::text) asc, created desc);

create trigger inc_threads
    after insert
    on threads
    for each row
execute procedure inc_params();

create table if not exists posts
(
    author varchar(128) not null,
    created timestamp not null,
    forum varchar(128) not null,
    id serial not null
        constraint posts_pk
            primary key,
    isedited boolean default false not null,
    message text not null,
    thread integer not null,
    parent integer,
    path integer[] not null
);

alter table posts owner to docker;

create index if not exists posts_created_thread_idx
    on posts (created, thread);

create index if not exists posts_cardinality_idx
    on posts (cardinality(path));

create index if not exists posts_thread_parent_idx
    on posts (thread, parent);

create trigger inc_posts
    after insert
    on posts
    for each row
execute procedure inc_params();

create trigger check_parent_tr
    before insert
    on posts
    for each row
execute procedure check_parent_thread();

create table if not exists votes_info
(
    votes boolean,
    thread_id integer not null,
    nickname varchar(128) not null,
    constraint only_one_voice
        unique (thread_id, nickname)
);

alter table votes_info owner to docker;

create trigger after_modify_votes
    after insert or update
    on votes_info
    for each row
execute procedure handler_data();

create trigger votes_to_bool
    before insert or update
    on votes_info
    for each row
execute procedure get_nickname();

create table if not exists user_forum
(
    forum varchar(128) not null,
    nickname varchar(128) not null,
    fullname varchar(128) not null,
    about text,
    email varchar(128) not null
);

alter table user_forum owner to docker;

create unique index if not exists user_forum_forum_lower_nickname_fullname_about_email_idx
    on user_forum (forum, lower(nickname::text), nickname, fullname, about, email);

create unique index if not exists user_forum_forum_lower_nickname_fullname_about_email_idx1
    on user_forum (forum asc, lower(nickname::text) desc, nickname asc, fullname asc, about asc, email asc);

