package handlers

import (
	"../models"
	"database/sql"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"net/http"
	"strconv"
	"time"
)

func (h *Handler) CreatePost(w http.ResponseWriter, r *http.Request) {
	postResult := []*models.Post{}
	json.NewDecoder(r.Body).Decode(&postResult)
	var queryPost string
	threadId, err := h.toID(r)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	//slugOrID := mux.Vars(r)["slug_or_id"]
	//threadId := -1
	//id, err := strconv.Atoi(slugOrID)
	//if err != nil {
	//	slug := slugOrID
	//	if err := h.DB.Get(&threadId, "select id from threads where slug=$1 limit 1", slug); err != nil {
	//		w.WriteHeader(http.StatusNotFound)
	//		return
	//	}
	//} else {
	//	threadId = id
	//}

	var forumSlug string
	if err := h.DB.Get(&forumSlug, `select forums.slug from forums
		inner join threads t on forums.slug = t.forum
	where t.id=$1 limit 1`, threadId); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	currTime := time.Now()
	tx := h.DB.MustBegin()
	for _, v := range postResult {
		v.Created = currTime
		v.Thread = threadId
		v.Forum = forumSlug
		if v.Parent.Int64 != 0 {
			queryPost = `insert into posts(author,created,forum,message,parent,thread) 
		values (:author,:created,:forum,:message,:parent,:thread)`
		} else {
			queryPost = `insert into posts(author,created,forum,message,thread) 
		values (:author,:created,:forum,:message,:thread) returning *`
		}
		//todo возвращет то что идет на вход, а нужно брать то что в бд создается
		if _, err := h.DB.NamedExec(queryPost, v); err != nil {
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
	//todo error null to int
	if err := h.DB.
		Select(&retPosts, `select * from posts where created=$1 order by id`, currTime); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(retPosts)
	w.WriteHeader(http.StatusCreated)

}
func (h *Handler) GetPost(w http.ResponseWriter, r *http.Request) {
	stringID := mux.Vars(r)["id"]
	postID, err := strconv.Atoi(stringID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	related, _ := r.URL.Query()["related"]
	retPost := models.Post{}
	//нету параметров
	if err := h.DB.Get(&retPost, `select * from posts where id=$1`, postID); err != nil {
		switch {
		case err == sql.ErrNoRows:
			w.WriteHeader(http.StatusNotFound)
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	thread := models.Thread{}
	forum := models.Forum{}
	author := models.User{}

	for _, v := range related {
		switch v {
		case "author":
			if err := h.DB.Get(&author, `select * from users where nickname=$1`, retPost.Author); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		case "forum":
			if err := h.DB.Get(&forum, `select * from forums where slug=$1`, retPost.Forum); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		case "thread":
			if err := h.DB.Get(&thread, `select * from threads where id=$1`, retPost.Thread); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}

	res := make(map[string]interface{})

	res["post"] = retPost

	if thread.Title != "" {
		res["thread"] = thread
	}
	if author.Email.String != "" {
		res["author"] = author
	}
	if forum.Slug != "" {
		res["forum"] = forum
	}

	json.NewEncoder(w).Encode(&res)

}
func (h *Handler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	stringID := mux.Vars(r)["id"]
	postID, err := strconv.Atoi(stringID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	retPost := models.Post{}
	newPostData := models.PostUpdate{}

	if err := json.NewDecoder(r.Body).Decode(&newPostData); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := h.DB.Get(&retPost, `select * from posts where id=$1`, postID); err != nil {
		switch {
		case err == sql.ErrNoRows:
			w.WriteHeader(http.StatusNotFound)
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	updatedPost := models.Post{}

	err = h.DB.Get(&updatedPost, `update posts set message=$1, isedited=true where id=$2 returning *`, newPostData.Message, postID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(&updatedPost)
}

func (h *Handler) GetAllPosts(w http.ResponseWriter, r *http.Request) {
	//взяли id треда
	id, err := h.toID(r)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	postList := []models.Post{}

	params := &models.PostParams{}
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	if err := decoder.Decode(params, r.URL.Query()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	query := `select * from posts where thread=$1 `

	if params.Since > 0 {
		query += ` and id>` + strconv.Itoa(params.Since)
	}

	if params.Desc {
		query += ` order by created desc `
	} else {
		query += ` order by created `
	}

	if params.Limit > 0 {
		query += ` limit ` + strconv.Itoa(params.Limit)
	}

	switch params.Sort {
	case "tree":
	case "parent_tree":
	default:
	}
	//todo sort флаг не учитывается
	err = h.DB.Select(&postList, query, id)

	json.NewEncoder(w).Encode(postList)
	return

}
func (h *Handler) toID(r *http.Request) (int, error) {
	slugOrID := mux.Vars(r)["slug_or_id"]
	threadId := -1
	id, err := strconv.Atoi(slugOrID)
	if err != nil {
		slug := slugOrID
		if err := h.DB.Get(&threadId, "select id from threads where slug=$1 limit 1", slug); err != nil {
			return -1, errors.New("not found")
		}
	} else {
		threadId = id
	}

	err = h.DB.Get(&threadId, "select id from threads where id=$1", threadId)
	if err == sql.ErrNoRows {
		return -1, errors.New("not found")
	}
	return threadId, nil
}
