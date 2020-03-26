create table if not exists users
(
    nickname text,
    fullname text not null,
    about text,
    email text not null
        constraint users_pk
            primary key
);

alter table users owner to andrey;

create unique index if not exists users_email_uindex
    on users (email);

create unique index if not exists users_nickname_uindex
    on users (nickname);

create table if not exists forums
(
    posts integer default 0 not null,
    slug text not null
        constraint forums_pk
            primary key,
    threads integer default 0 not null,
    title text not null,
    "user" text not null
        constraint forums_users_nickname_fk
            references users (nickname)
);

alter table forums owner to andrey;

create unique index if not exists forums_slug_uindex
    on forums (slug);

create table if not exists threads
(
    author text not null
        constraint threads_users_nickname_fk
            references users (nickname),
    created timestamp default CURRENT_TIMESTAMP,
    forum text
        constraint threads_forums_slug_fk
            references forums
            on update cascade on delete cascade,
    id serial not null
        constraint threads_pk
            primary key,
    message text not null,
    slug text not null,
    title text not null,
    votes integer default 0 not null
);

alter table threads owner to andrey;

create unique index if not exists threads_id_uindex
    on threads (id);

create table if not exists posts
(
    author text not null
        constraint posts_users_nickname_fk
            references users (nickname),
    created timestamp default CURRENT_TIMESTAMP,
    forum text not null
        constraint posts_forums_slug_fk
            references forums,
    id serial not null
        constraint posts_pk
            primary key,
    isedited boolean default false not null,
    message text not null,
    parent integer default 0 not null
        constraint posts_posts_id_fk
            references posts
            on update cascade on delete cascade,
    thread integer not null
        constraint posts_threads_id_fk
            references threads
            on update cascade on delete cascade
);

alter table posts owner to andrey;

