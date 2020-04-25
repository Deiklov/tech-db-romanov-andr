package main

import (
	"github.com/Deiklov/tech-db-romanov-andr/golang/handlers"
	"github.com/Deiklov/tech-db-romanov-andr/golang/middleware"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"
	"log"
	"net/http"
)

func main() {
	router := mux.NewRouter()
	//connectionString := "dbname=docker user=docker password=docker host=0.0.0.0 port=5432"
	connectionString := "dbname=tmpxx user=andrey password=167839 host=localhost port=5432"
	db, err := sqlx.Connect("pgx", connectionString)
	if err != nil {
		log.Fatal(err)
	}
	conf := pgx.ConnPoolConfig{
		ConnConfig: pgx.ConnConfig{
			Host:     "localhost",
			User:     "andrey",
			Database: "tmpxx",
			Password: "167839",
			Port:     5432,
		},
		MaxConnections: 20,
	}
	conn, err := pgx.NewConnPool(conf)
	if err != nil {
		panic(err)
	}
	//createDB(db)

	Handlers := handlers.Handler{db, conn}
	r := router.PathPrefix("/api").Subrouter()

	r.HandleFunc("/user/{nickname}/create", Handlers.CreateUser).Methods(http.MethodPost)
	r.HandleFunc("/user/{nickname}/profile", Handlers.UpdateUser).Methods(http.MethodPost)
	r.HandleFunc("/user/{nickname}/profile", Handlers.GetUser).Methods(http.MethodGet)

	r.HandleFunc("/forum/create", Handlers.CreateForum).Methods(http.MethodPost)
	r.HandleFunc("/forum/{slug}/details", Handlers.ForumDetails).Methods(http.MethodGet)
	r.HandleFunc("/forum/{slug}/create", Handlers.NewThread).Methods(http.MethodPost)
	r.HandleFunc("/forum/{slug}/threads", Handlers.AllThreadsFromForum).Methods(http.MethodGet)
	r.HandleFunc("/forum/{slug}/users", Handlers.AllUsersForum).Methods(http.MethodGet)

	r.HandleFunc("/thread/{slug_or_id}/details", Handlers.ThreadInfo).Methods(http.MethodGet)
	r.HandleFunc("/thread/{slug_or_id}/details", Handlers.ThreadUpdate).Methods(http.MethodPost)
	r.HandleFunc("/thread/{slug_or_id}/vote", Handlers.ThreadVotes).Methods(http.MethodPost)

	r.HandleFunc("/service/clear", Handlers.ServiceClear).Methods(http.MethodPost)
	r.HandleFunc("/service/status", Handlers.ServiceInfo).Methods(http.MethodGet)

	r.HandleFunc("/thread/{slug_or_id}/create", Handlers.CreatePost).Methods(http.MethodPost)
	r.HandleFunc("/thread/{slug_or_id}/posts", Handlers.GetAllPosts).Methods(http.MethodGet)
	r.HandleFunc("/post/{id}/details", Handlers.UpdatePost).Methods(http.MethodPost)
	r.HandleFunc("/post/{id}/details", Handlers.GetPost).Methods(http.MethodGet)
	http.Handle("/", r)

	if err := http.ListenAndServe(":5000", middleware.SetApplJson(r)); err != nil {
		log.Fatal(err)
	}
}
func createDB(db *sqlx.DB) {
	_, err := db.Exec(`
CREATE EXTENSION IF NOT EXISTS citext;
create table if not exists users
(
    nickname text   not null
        constraint users_pk
            primary key,
    fullname text   not null,
    about    text,
    email    citext not null
);

alter table users
    owner to docker;

create unique index if not exists users_lower_idx
    on users (lower(nickname));

create unique index if not exists users_email_uindex
    on users (email);

create table if not exists forums
(
    posts   integer default 0 not null,
    slug    text              not null
        constraint forums_pk
            primary key,
    threads integer default 0 not null,
    title   text              not null,
    "user"  text              not null
        constraint forums_users_nickname_fk
            references users
            on update set null on delete set null
);

alter table forums
    owner to docker;

create index if not exists forums_lower_idx
    on forums (lower("user"));

create unique index if not exists forums_lower_idx1
    on forums (lower(slug));

create table if not exists threads
(
    author  text                not null
        constraint threads_users_nickname_fk
            references users,
    created timestamp default CURRENT_TIMESTAMP,
    forum   text
        constraint threads_forums_slug_fk
            references forums
            on update cascade on delete cascade,
    id      serial              not null
        constraint threads_pk
            primary key,
    message text                not null,
    slug    text,
    title   text                not null,
    votes   integer   default 0 not null
);

alter table threads
    owner to docker;

create unique index if not exists threads_slug_uindex
    on threads (lower(slug));


create table if not exists posts
(
    author   text                    not null
        constraint posts_users_nickname_fk
            references users,
    created  timestamp default CURRENT_TIMESTAMP,
    forum    text                    not null
        constraint posts_forums_slug_fk
            references forums
            on update cascade on delete cascade,
    id       serial                  not null
        constraint posts_pk
            primary key,
    isedited boolean   default false not null,
    message  text                    not null,
    parent   integer
        constraint posts_posts_id_fk
            references posts
            on update cascade on delete cascade,
    thread   integer                 not null
        constraint posts_threads_id_fk
            references threads
            on update cascade on delete cascade
);

alter table posts
    owner to docker;

create index if not exists posts_lower_idx
    on posts (lower(author));

create index if not exists posts_lower_idx1
    on posts (lower(forum));

create index if not exists posts_thread_idx
    on posts (thread);

create index if not exists posts_parent_idx
    on posts (parent);


create table if not exists votes_info
(
    votes     boolean,
    thread_id integer not null
        constraint votes_info_threads_id_fk
            references threads
            on update cascade on delete cascade,
    nickname  text    not null
        constraint votes_info_users_nickname_fk
            references users,
    constraint only_one_voice
        unique (thread_id, nickname)
);

alter table votes_info
    owner to docker;

create index if not exists votes_info_lower_idx
    on votes_info (lower(nickname));

create index if not exists votes_info_thread_id_idx
    on votes_info (thread_id);

CREATE OR REPLACE FUNCTION check_parent_thread() returns trigger
    language plpgsql
as
$$
DECLARE
    i int2;
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
    end if;

    RETURN NEW;
END;
$$;

alter function check_parent_thread() owner to docker;

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
        update forums set threads=threads + 1 where slug = forum_slug;
    elsif tg_name = 'inc_posts' then
        update forums set posts=posts + 1 where slug = forum_slug;
    end if;
    return new;
end;
$$;

alter function inc_params() owner to docker;

create or replace function voice_to_bool() returns trigger
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

alter function voice_to_bool() owner to docker;






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

create trigger votes_to_bool
    before insert or update
    on votes_info
    for each row
execute procedure voice_to_bool();

create trigger after_modify_votes
    after insert or update
    on votes_info
    for each row
execute procedure handler_data();

create trigger inc_threads
    after insert
    on threads
    for each row
execute procedure inc_params();
`)

	if err != nil {
		log.Fatal(err)
	}
}
