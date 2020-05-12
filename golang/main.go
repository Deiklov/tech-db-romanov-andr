package main

import (
	"context"
	"github.com/Deiklov/tech-db-romanov-andr/golang/handlers"
	"github.com/Deiklov/tech-db-romanov-andr/golang/middleware"
	"github.com/fasthttp/router"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jmoiron/sqlx"
	"github.com/valyala/fasthttp"
	"io/ioutil"
	"log"
)

func main() {
	r := router.New()
	connectionString := "dbname=docker user=docker password=docker host=0.0.0.0 port=5432"
	//connectionString := "dbname=db_forum user=andrey password=167839 host=localhost port=5432"
	db, err := sqlx.Connect("pgx", connectionString)
	if err != nil {
		log.Fatal(err)
	}
	conn, err := pgxpool.Connect(context.Background(), connectionString)

	if err != nil {
		panic(err)
	}
	createDB(conn)

	Handlers := handlers.Handler{db, conn}

	r.POST("/api/user/{nickname}/create", middleware.SetJson(Handlers.CreateUser))
	r.POST("/api/user/{nickname}/profile", middleware.SetJson(Handlers.UpdateUser))
	r.GET("/api/user/{nickname}/profile", middleware.SetJson(Handlers.GetUser))

	r.POST("/api/forum/create", middleware.SetJson(Handlers.CreateForum))
	r.GET("/api/forum/{slug}/details", middleware.SetJson(Handlers.ForumDetails))
	r.POST("/api/forum/{slug}/create", middleware.SetJson(Handlers.NewThread))

	r.GET("/api/forum/{slug}/threads", middleware.SetJson(Handlers.AllThreadsFromForum))
	r.GET("/api/forum/{slug}/users", middleware.SetJson(Handlers.AllUsersForum))

	r.GET("/api/thread/{slug_or_id}/details", middleware.SetJson(Handlers.ThreadInfo))
	r.POST("/api/thread/{slug_or_id}/details", middleware.SetJson(Handlers.ThreadUpdate))
	r.POST("/api/thread/{slug_or_id}/vote", middleware.SetJson(Handlers.ThreadVotes))

	r.POST("/api/service/clear", middleware.SetJson(Handlers.ServiceClear))
	r.GET("/api/service/status", middleware.SetJson(Handlers.ServiceInfo))

	r.POST("/api/thread/{slug_or_id}/create", middleware.SetJson(Handlers.CreatePost))
	r.GET("/api/thread/{slug_or_id}/posts", middleware.SetJson(Handlers.GetAllPosts))
	r.POST("/api/post/{id}/details", middleware.SetJson(Handlers.UpdatePost))
	r.GET("/api/post/{id}/details", middleware.SetJson(Handlers.GetPost))

	if err := fasthttp.ListenAndServe(":5000", r.Handler); err != nil {
		log.Fatal(err)
	}
}
func createDB(conn *pgxpool.Pool) {
	data, err := ioutil.ReadFile("/usr/bin/functions.sql")
	if err != nil {
		log.Fatal(err)
	}
	_, err = conn.Exec(context.Background(), string(data))
	if err != nil {
		log.Fatal(err)
	}
	data, err = ioutil.ReadFile("/usr/bin/database.sql")
	if err != nil {
		log.Fatal(err)
	}
	_, err = conn.Exec(context.Background(), string(data))
	if err != nil {
		log.Fatal(err)
	}
}
