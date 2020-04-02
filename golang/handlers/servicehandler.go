package handlers

import (
	"../models"
	"encoding/json"
	"net/http"
)

func (handler *Handler) ServiceInfo(w http.ResponseWriter, r *http.Request) {
	serviceInfo := &models.Info{}
	forumsQuery := `select count(*) forum from forums`
	usersQuery := `select count(*) "user" from users`
	threadsQuery := `select count(*) thread from threads`
	postsQuery := `select count(*) post from posts`
	if err := handler.DB.Get(serviceInfo, forumsQuery); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := handler.DB.Get(serviceInfo, usersQuery); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := handler.DB.Get(serviceInfo, postsQuery); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := handler.DB.Get(serviceInfo, threadsQuery); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(serviceInfo)
}

func (handler *Handler) ServiceClear(w http.ResponseWriter, r *http.Request) {
	stmtdelete := `TRUNCATE users,forums,posts,threads restart identity cascade`
	if _, err := handler.DB.Exec(stmtdelete); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
