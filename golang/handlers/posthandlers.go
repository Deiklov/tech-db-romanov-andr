package handlers

import (
	"../models"
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
		}
	}
	if err := tx.Commit(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(postResult)
	w.WriteHeader(http.StatusCreated)

}
func (handler *Handler) GetPost(w http.ResponseWriter, r *http.Request) {

}
func (handler *Handler) UpdatePost(w http.ResponseWriter, r *http.Request) {

}
func (handler *Handler) GetAllPosts(w http.ResponseWriter, r *http.Request) {

}
