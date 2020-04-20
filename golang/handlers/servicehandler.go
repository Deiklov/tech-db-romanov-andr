package handlers

import (
	"github.com/Deiklov/tech-db-romanov-andr/golang/models"
	"github.com/mailru/easyjson"
	"net/http"
)

func (h *Handler) ServiceInfo(w http.ResponseWriter, r *http.Request) {
	serviceInfo := &models.Info{}
	forumsQuery := `select count(*) forum from forums`
	usersQuery := `select count(*) "user" from users`
	threadsQuery := `select count(*) thread from threads`
	postsQuery := `select count(*) post from posts`
	if err := h.DB.Get(serviceInfo, forumsQuery); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := h.DB.Get(serviceInfo, usersQuery); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := h.DB.Get(serviceInfo, postsQuery); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := h.DB.Get(serviceInfo, threadsQuery); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, _, err := easyjson.MarshalToHTTPResponseWriter(serviceInfo, w); err != nil {
		http.Error(w, "easy", 500)
		return
	}
}

func (h *Handler) ServiceClear(w http.ResponseWriter, r *http.Request) {
	stmtdelete := `TRUNCATE users,forums,posts,threads restart identity cascade`
	if _, err := h.DB.Exec(stmtdelete); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
