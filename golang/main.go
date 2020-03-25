package main

import (
	"./middleware"
	"./userhandlers"
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
	userhandler := userhandlers.UserHandler{db}
	r.HandleFunc("/user/{nickname}/create", userhandler.CreateUser).Methods(http.MethodPost)
	r.HandleFunc("/user/{nickname}/profile", userhandler.UpdateUser).Methods(http.MethodPost)
	r.HandleFunc("/user/{nickname}/profile", userhandler.GetUser).Methods(http.MethodGet)
	http.Handle("/", r)
	http.ListenAndServe(":8080", middleware.SetApplJson(r))
}
