package handlers

import (
	"../models"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"net/http"
)

func (handler *Handler) CreateForum(w http.ResponseWriter, r *http.Request) {
	newForum := &models.Forum{}
	if err := json.NewDecoder(r.Body).Decode(newForum); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Invalid json !"}`))
		return
	}
	//чекаем есть ли юзер
	queryUser := `select nickname from users where nickname=$1;`
	row := handler.DB.QueryRow(queryUser, newForum.UserNick)
	var nickname string
	row.Scan(&nickname)
	//если нет юзера, то кидаем 404
	if nickname == "" {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Can't find user with that nickname"})
		return
	}
	//пытаемся инсертнуть
	queryForum := `insert into forums (slug, title,user) values($1,$2,$3) returning slug;`
	row = handler.DB.QueryRow(queryForum, newForum.Slug, newForum.Title, newForum.UserNick)
	//сканим ответ
	var backSlug string
	err := row.Scan(&backSlug)
	//если ошибка при инсерте, то по ветке 409 идем
	if err, ok := err.(*pq.Error); ok {
		switch err.Code {
		//this is conflict code
		case "23505":
			w.WriteHeader(http.StatusConflict)
			oldForum := &models.Forum{}
			userInsertState := `SELECT * from forums where slug=$1;`
			row := handler.DB.QueryRow(userInsertState, newForum.Slug)
			row.Scan(&oldForum)
			json.NewEncoder(w).Encode(oldForum)
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Some error with data querys!"))
			return
		}
	}
	//если все гуд то 201 долбанем
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newForum)
}

func (handler *Handler) ForumDetails(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]
	queryForum := `select * from forums where slug=$1`
	row := handler.DB.QueryRow(queryForum, slug)
	foundForum := &models.Forum{}
	err := row.Scan(&foundForum)
	//можно проверить любое поле на пустоту
	if foundForum.Slug == "" {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Can't find forum with this slug"})
		return
	}
	//если нашли его, то ретурним
	if err == nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(foundForum)
		return
	}
}

func (handler *Handler) NewThread(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]
	//todo может ошибка выпрыгнуть при декодере
	//todo slug может быть не задан
	newThrd := &models.Thread{Slug: slug}
	if err := json.NewDecoder(r.Body).Decode(newThrd); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Invalid json !"}`))
		return
	}
	queryThreads := `insert into threads (author, forum, message, slug, title) values($1,$2,$3,$4,$5) returning id;`
	row := handler.DB.QueryRow(queryThreads, newThrd.Author, newThrd.Forum, newThrd.Message, newThrd.Slug, newThrd.Title)
	returningID := new(int)
	if err := row.Scan(&returningID); err != nil {
		if err, ok := err.(*pq.Error); ok {
			switch err.Code {
			//не вставит если нет юзера или форума
			case "23503":
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]string{"message": "not exsist that user or forum"})
				return
				//если хрен знает какая ошибка
			default:
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Some error with data querys!"))
				return
			}
		}
	}

}
