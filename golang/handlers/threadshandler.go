package handlers

import (
	"database/sql"
	"github.com/Deiklov/tech-db-romanov-andr/golang/models"
	"github.com/jackc/pgx"
	"github.com/mailru/easyjson"
	"net/http"
	"strconv"
)

func (h *Handler) ThreadInfo(w http.ResponseWriter, r *http.Request) {
	thread := &models.Thread{}
	id, err := h.toID(r)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		easyjson.MarshalToHTTPResponseWriter(models.NotFoundMsg, w)
		return
	}

	if err := h.DB.Get(thread, `SELECT * from threads where id=$1`, id); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, _, err := easyjson.MarshalToHTTPResponseWriter(thread, w); err != nil {
		http.Error(w, "easy", 500)
		return
	}
}

func (h *Handler) ThreadUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := h.toID(r)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		easyjson.MarshalToHTTPResponseWriter(models.NotFoundMsg, w)
		return
	}
	threadUPD := &models.ThreadUpdate{}
	threadResult := &models.Thread{}
	if err := easyjson.UnmarshalFromReader(r.Body, threadUPD); err != nil {
		http.Error(w, "easy", 500)
		return
	}

	queryThread := `update threads set`
	if threadUPD.Message != "" {
		queryThread += ` message='` + threadUPD.Message + `' `
	}
	if threadUPD.Title != "" {
		if threadUPD.Message != "" {
			queryThread += `,`
		}
		queryThread += ` title='` + threadUPD.Title + `' `
	}
	queryThread += ` where id=` + strconv.Itoa(id)

	if threadUPD.Message == "" && threadUPD.Title == "" {
		h.DB.Get(threadResult, `select * from threads where id=$1`, id)
		if _, _, err := easyjson.MarshalToHTTPResponseWriter(threadResult, w); err != nil {
			http.Error(w, "easy", 500)
			return
		}
		return
	}

	queryThread += ` returning *;`
	err = h.DB.Get(threadResult, queryThread)
	if err != nil {
		if threadResult.Id <= 0 {
			w.WriteHeader(http.StatusNotFound)
			easyjson.MarshalToHTTPResponseWriter(models.NotFoundMsg, w)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, _, err := easyjson.MarshalToHTTPResponseWriter(threadResult, w); err != nil {
		http.Error(w, "easy", 500)
		return
	}
}

func (h *Handler) ThreadVotes(w http.ResponseWriter, r *http.Request) {
	voice := &models.Vote{}
	threadResult := &models.Thread{}
	easyjson.UnmarshalFromReader(r.Body, voice)

	threadID, err := h.toID(r)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		easyjson.MarshalToHTTPResponseWriter(models.NotFoundMsg, w)
		return
	}

	var voiceBool bool
	if voice.Voice == 1 {
		voiceBool = true
	}

	query := `insert into votes_info(votes, thread_id, nickname)
VALUES ($1, $2, $3)
on conflict on constraint only_one_voice do update set votes=excluded.votes
returning *`

	_, err = h.DB.Exec(query, voiceBool, threadID, voice.Nickname)
	if err, ok := err.(pgx.PgError); ok {
		switch err.Code {
		case "23502":
			w.WriteHeader(http.StatusNotFound)
			easyjson.MarshalToHTTPResponseWriter(models.NotFoundMsg, w)
			return
		}
	}

	err = h.DB.Get(threadResult, `select * from threads where id=$1`, threadID)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		easyjson.MarshalToHTTPResponseWriter(models.NotFoundMsg, w)
		return
	}

	if _, _, err := easyjson.MarshalToHTTPResponseWriter(threadResult, w); err != nil {
		http.Error(w, "easyjson err", 500)
		return
	}
}
