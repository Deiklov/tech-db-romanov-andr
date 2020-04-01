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
	handlers := handlers.Handler{db}
	r.HandleFunc("/user/{nickname}/create", handlers.CreateUser).Methods(http.MethodPost)
	r.HandleFunc("/user/{nickname}/profile", handlers.UpdateUser).Methods(http.MethodPost)
	r.HandleFunc("/user/{nickname}/profile", handlers.GetUser).Methods(http.MethodGet)

	r.HandleFunc("/forum/create", handlers.CreateForum).Methods(http.MethodPost)
	r.HandleFunc("/forum/{slug}/details", handlers.ForumDetails).Methods(http.MethodGet)
	r.HandleFunc("/forum/{slug}/create", handlers.NewThread).Methods(http.MethodPost)
	r.HandleFunc("/forum/{slug}/threads", handlers.AllThreadsFromForum).Methods(http.MethodGet)
	r.HandleFunc("/forum/{slug}/users", handlers.AllUsersForum).Methods(http.MethodGet)
	http.Handle("/", r)
	http.ListenAndServe(":5000", middleware.SetApplJson(r))
}
