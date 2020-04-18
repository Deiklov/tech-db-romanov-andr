package handlers

import (
	"database/sql"
	"encoding/json"
	"github.com/Deiklov/tech-db-romanov-andr/golang/models"
	"github.com/lib/pq"
	"net/http"
	"strconv"
)

func (h *Handler) ThreadInfo(w http.ResponseWriter, r *http.Request) {
	thread := &models.Thread{}
	id, err := h.toID(r)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "not found this threads"})
		return
	}

	if err := h.DB.Get(thread, `SELECT * from threads where id=$1`, id); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(thread)
}

func (h *Handler) ThreadUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := h.toID(r)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": `not found this threads` + err.Error() + ``})
		return
	}
	threadUPD := &models.ThreadUpdate{}
	threadResult := &models.Thread{}
	json.NewDecoder(r.Body).Decode(&threadUPD)

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
		json.NewEncoder(w).Encode(threadResult)
		return
	}

	queryThread += ` returning *;`
	err = h.DB.Get(threadResult, queryThread)
	if err != nil {
		if threadResult.Id <= 0 {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"message": "not found this threads"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(threadResult)
}

func (h *Handler) ThreadVotes(w http.ResponseWriter, r *http.Request) {
	voice := &models.Vote{}
	threadResult := &models.Thread{}
	json.NewDecoder(r.Body).Decode(voice)
	threadID, err := h.toID(r)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "not found this threads"})
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
	if err, ok := err.(*pq.Error); ok {
		switch err.Code {
		case "23502":
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"message": "Can't find user with that nickname"})
			return
		}
	}

	err = h.DB.Get(threadResult, `select * from threads where id=$1`, threadID)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "not found this threads"})
		return
	}

	json.NewEncoder(w).Encode(threadResult)
}
