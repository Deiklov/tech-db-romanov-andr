package main

import (
	"github.com/Deiklov/tech-db-romanov-andr/golang/handlers"
	"github.com/Deiklov/tech-db-romanov-andr/golang/middleware"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"log"
	"net/http"
)

func main() {
	router := mux.NewRouter()
	connectionString := "dbname=docker user=docker password=docker host=0.0.0.0 port=5432"
	db, err := sqlx.Open("postgres", connectionString)

	if err != nil {
		log.Fatal(err)
	}

	createDB(db)

	Handlers := handlers.Handler{db}
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
	_, err := db.Exec(`create table if not exists users
(
    nickname text not null,
    fullname text not null,
    about text,
    email text not null
        constraint users_pk
            primary key
);

alter table users owner to docker;

create unique index if not exists users_nickname_uindex
    on users (nickname);

create unique index if not exists user_email_uindex
    on users (lower(email));

create unique index if not exists user_nickname_uindex
    on users (lower(nickname));

create unique index if not exists user_nickname_index
    on users (lower(nickname));

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

alter table forums owner to docker;

create unique index if not exists forums_slug_uindex
    on forums (slug);

create unique index if not exists forum_slug_index
    on forums (lower(slug));

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
    slug text,
    title text not null,
    votes integer default 0 not null
);
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

alter table threads owner to docker;

create unique index if not exists threads_id_uindex
    on threads (id);

create unique index if not exists threads_slug_uindex
    on threads (lower(slug));

create table if not exists posts
(
    author text not null
        constraint posts_users_nickname_fk
            references users (nickname),
    created timestamp default CURRENT_TIMESTAMP,
    forum text not null
        constraint posts_forums_slug_fk
            references forums
            on update cascade on delete cascade,
    id serial not null
        constraint posts_pk
            primary key,
    isedited boolean default false not null,
    message text not null,
    parent integer
        constraint posts_posts_id_fk
            references posts
            on update cascade on delete cascade,
    thread integer not null
        constraint posts_threads_id_fk
            references threads
            on update cascade on delete cascade
);

alter table posts owner to docker;

create trigger inc_posts
    after insert
    on posts
    for each row
execute procedure inc_params();

create table if not exists votes_info
(
    votes integer,
    thread_id integer not null
        constraint votes_info_threads_id_fk
            references threads
            on update cascade on delete cascade,
    nickname text not null
        constraint votes_info_users_nickname_fk
            references users (nickname)
            on update cascade on delete cascade
);

alter table votes_info owner to docker;

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
