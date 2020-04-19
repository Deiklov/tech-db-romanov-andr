package handlers

import (
	"database/sql"
	"encoding/json"
	"github.com/Deiklov/tech-db-romanov-andr/golang/models"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/lib/pq"
	"github.com/mailru/easyjson"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (h *Handler) CreateForum(w http.ResponseWriter, r *http.Request) {
	newForum := &models.Forum{}
	if err := easyjson.UnmarshalFromReader(r.Body, newForum); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//чекаем есть ли юзер
	var nickname string
	err := h.DB.QueryRow(`select nickname from users where lower(nickname)=lower($1);`, newForum.UserNick).Scan(&nickname)
	//если нет юзера, то кидаем 404
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Can't find user with that nickname"})
		return
	}

	newForum.UserNick = nickname
	backSlug := ""

	queryForum := `insert into forums (slug, title,"user") values($1,$2,$3) returning slug;`
	err = h.DB.Get(&backSlug, queryForum, newForum.Slug, newForum.Title, newForum.UserNick)

	if err, ok := err.(*pq.Error); ok {
		switch err.Code {
		//this is conflict code
		case "23505":
			w.WriteHeader(http.StatusConflict)
			oldForum := &models.Forum{}

			userInsertState := `SELECT * from forums where lower(slug)=lower($1);`
			if err := h.DB.Get(oldForum, userInsertState, newForum.Slug); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("конфликт, не смог выбрать форум"))
				return
			}

			if _, _, err := easyjson.MarshalToHTTPResponseWriter(oldForum, w); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Some error with data querys!"))
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
	//json.NewEncoder(w).Encode(newForum)
	if _, _, err := easyjson.MarshalToHTTPResponseWriter(newForum, w); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func (h *Handler) ForumDetails(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]
	foundForum := &models.Forum{}
	err := h.DB.Get(foundForum, `select * from forums where lower(slug)=lower($1)`, slug)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Can't find forum with this slug"})
		return
	}

	if err == nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(foundForum)
		return
	}
}

func (h *Handler) NewThread(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]
	forumDB := models.Forum{}
	authorDB := models.User{}

	newThrd := &models.Thread{}
	if err := json.NewDecoder(r.Body).Decode(newThrd); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Invalid json !"}`))
		return
	}

	err := h.DB.Get(&forumDB, `select * from forums where lower(slug)=lower($1)`, slug)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "not found forum for this slug"})
		return
	}

	err = h.DB.Get(&authorDB, `select * from users where lower(nickname)=lower($1)`, newThrd.Author)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "not found  this author"})
		return
	}

	newThrd.Author = authorDB.Nickname
	newThrd.Forum = forumDB.Slug
	newThrd.Created = newThrd.Created.UTC()
	queryThreads := `insert into threads (author, forum, message, title,created) values($1,$2,$3,$4,$5) returning *;`
	if newThrd.Slug.Valid {
		queryThreads = `insert into threads (author, forum, message, title,created,slug) values($1,$2,$3,$4,$5,'` +
			newThrd.Slug.String + `') returning *;`
	}
	err = h.DB.Get(newThrd, queryThreads, newThrd.Author, newThrd.Forum, newThrd.Message, newThrd.Title, newThrd.Created)

	if err != nil {
		if err, ok := err.(*pq.Error); ok {
			switch err.Code {
			//не вставит если нет юзера или форума
			case "23503":
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]string{"message": "not exsist that user or forum"})
				return
			case "23505":
				w.WriteHeader(http.StatusConflict)
				exsistThread := models.Thread{}
				h.DB.Get(&exsistThread, `select * from threads where lower(slug)=lower($1)`, newThrd.Slug.String)
				json.NewEncoder(w).Encode(exsistThread)
				return
			default:
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Some error with data querys!"))
				return
			}
		}
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newThrd)
}

func (h *Handler) AllThreadsFromForum(w http.ResponseWriter, r *http.Request) {
	params := &models.ThreadParams{}
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	if err := decoder.Decode(params, r.URL.Query()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	slug := mux.Vars(r)["slug"]

	items := []models.Thread{}
	params.Since = params.Since.UTC()

	forumSlugFromDB := ""
	err := h.DB.Get(&forumSlugFromDB, `select slug from forums where lower(slug)=lower($1)`, slug)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "not found"})
		return
	}

	threadsQuery := `SELECT author,created,forum,id,message,slug,title,votes
from threads where lower(forum)=lower($1) `

	if params.Desc {
		zeroTime := time.Time{}
		//если не указали время то при деск created нужен только для соответствия call(2 args)
		if params.Since == zeroTime {
			threadsQuery += ` and created >=$2 order by created desc `
		} else {
			threadsQuery += ` and created <=$2 order by created desc `
		}

	} else {
		threadsQuery += ` and created >=$2 order by created `
	}

	if params.Limit > 0 {
		threadsQuery += `limit ` + strconv.Itoa(params.Limit)
	}

	err = h.DB.Select(&items, threadsQuery, slug, params.Since)

	if err == sql.ErrNoRows {
		json.NewEncoder(w).Encode(items)
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	if err := json.NewEncoder(w).Encode(items); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *Handler) AllUsersForum(w http.ResponseWriter, r *http.Request) {
	params := &models.ForumUserParams{}
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	if err := decoder.Decode(params, r.URL.Query()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	slug := mux.Vars(r)["slug"]

	forumSlug := ""
	err := h.DB.Get(&forumSlug, `select slug from forums where lower(slug)=lower($1)`, slug)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "not found"})
		return
	}

	users := []models.User{}

	userQuery := `SELECT * FROM (select distinct about,email,fullname,nickname from threads 
    join users u on lower(threads.author) = lower(u.nickname) where lower(forum)=lower($1)
	UNION 
	SELECT DISTINCT about,email,fullname,nickname FROM posts 
	    JOIN users u2 on lower(posts.author) = lower(u2.nickname) WHERE lower(forum)=lower($1)) sub`

	if params.Desc {
		if params.Since != "" {
			userQuery += ` where lower(nickname)<'` + strings.ToLower(params.Since) + `' order by lower(nickname) desc `
		} else {
			userQuery += ` where lower(nickname)>'` + strings.ToLower(params.Since) + `' order by lower(nickname) desc `
		}

	} else {
		userQuery += ` where lower(nickname)>'` + strings.ToLower(params.Since) + `' order by lower(nickname)  `
	}

	if params.Limit > 0 {
		userQuery += ` limit ` + strconv.Itoa(params.Limit)
	}
	err = h.DB.Select(&users, userQuery, slug)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error with select already exsist user"))
		return
	}

	json.NewEncoder(w).Encode(users)
}
