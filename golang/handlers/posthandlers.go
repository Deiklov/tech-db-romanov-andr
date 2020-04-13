package handlers

import (
	"../models"
	"database/sql"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"net/http"
	"strconv"
	"time"
)

func (handler *Handler) CreatePost(w http.ResponseWriter, r *http.Request) {
	slugOrID := mux.Vars(r)["slug_or_id"]
	postResult := []*models.Post{}
	json.NewDecoder(r.Body).Decode(&postResult)
	var queryPost string
	threadId := -1
	id, err := strconv.Atoi(slugOrID)
	if err != nil {
		slug := slugOrID
		if err := handler.DB.Get(&threadId, "select id from threads where slug=$1 limit 1", slug); err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
	} else {
		threadId = id
	}

	var forumSlug string
	if err := handler.DB.Get(&forumSlug, `select forums.slug from forums
		inner join threads t on forums.slug = t.forum
	where t.id=$1 limit 1`, threadId); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	currTime := time.Now()
	tx := handler.DB.MustBegin()
	for _, v := range postResult {
		v.Created = currTime
		v.Thread = threadId
		v.Forum = forumSlug
		if v.Parent != 0 {
			queryPost = `insert into posts(author,created,forum,message,parent,thread) 
		values (:author,:created,:forum,:message,:parent,:thread)`
		} else {
			queryPost = `insert into posts(author,created,forum,message,thread) 
		values (:author,:created,:forum,:message,:thread) returning *`
		}
		//todo возвращет то что идет на вход, а нужно брать то что в бд создается
		if _, err := handler.DB.NamedExec(queryPost, v); err != nil {
			//откатим транзакцию
			if err := tx.Rollback(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			//вернем 409 код
			if err, ok := err.(*pq.Error); ok {
				switch err.Code {
				case "23503":
					w.WriteHeader(http.StatusConflict)
					return
				default:
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	retPosts := []*models.Post{}
	if err := handler.DB.
		Select(&retPosts, `select * from posts where created=$1 order by id`, currTime); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(retPosts)
	w.WriteHeader(http.StatusCreated)

}
func (handler *Handler) GetPost(w http.ResponseWriter, r *http.Request) {
	stringID := mux.Vars(r)["id"]
	postID, err := strconv.Atoi(stringID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	related, ok := r.URL.Query()["related"]
	retPost := models.Post{}
	if !ok || len(related[0]) < 1 {
		//нету параметров
		if err := handler.DB.Get(&retPost, `select * from posts where id=$1`, postID); err != nil {
			switch {
			case err == sql.ErrNoRows:
				w.WriteHeader(http.StatusNotFound)
				return
			default:
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		return
	}

	thread := models.Thread{}
	forum := models.Forum{}
	user := models.User{}

	for _, v := range related {
		switch v {
		case "author":
			if err := handler.DB.Get(&user, `select * from users where nickname=$1`, retPost.Author); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		case "forum":
			if err := handler.DB.Get(&forum, `select * from forums where slug=$1`, retPost.Forum); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		case "thread":
			if err := handler.DB.Get(&thread, `select * from threads where id=$1`, retPost.Thread); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}
	//var res []interface{}
	//if thr

}
func (handler *Handler) UpdatePost(w http.ResponseWriter, r *http.Request) {

}
func (handler *Handler) GetAllPosts(w http.ResponseWriter, r *http.Request) {

}
