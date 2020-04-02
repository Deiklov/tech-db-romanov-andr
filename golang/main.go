package main

import (
	"./handlers"
	"./middleware"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"log"
	"net/http"
)

func main() {
	r := mux.NewRouter()
	connectionString := "dbname=homework user=andrey password=167839 host=localhost port=5432"
	db, err := sqlx.Open("postgres", connectionString)
	if err != nil {
		log.Fatal(err)
	}
	Handlers := handlers.Handler{db}
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
