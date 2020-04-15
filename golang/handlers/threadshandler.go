package handlers

import (
	"../models"
	"database/sql"
	"encoding/json"
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

	votesInfo := models.VotesInfo{}

	err = h.DB.Get(&votesInfo, `select * from votes_info where nickname=$1 and thread_id=$2`, voice.Nickname, threadID)
	if err == nil {
		_, err := h.DB.Exec(
			`update votes_info set votes=$1 where nickname=$2 and thread_id=$3`,
			voice.Voice, votesInfo.Nickname, votesInfo.ThreadID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		switch votesInfo.Votes {
		case 1:
			if voice.Voice == 1 {
				voice.Voice = 0
			} else {
				voice.Voice = -2
			}
		case -1:
			if voice.Voice == -1 {
				voice.Voice = 0
			} else {
				voice.Voice = 2
			}
		}
	} else {
		_, err = h.DB.Exec(
			`insert into votes_info (votes,thread_id,nickname) values ($1,$2,$3)`,
			voice.Voice, threadID, voice.Nickname)
		if err, ok := err.(*pq.Error); ok {
			switch err.Code {
			//this is conflict code
			case "23503":
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]string{"message": "not found this user"})
				return
			default:
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}

	err = h.DB.Get(threadResult,
		`update threads set votes=votes+($1) where id=$2 returning *`, voice.Voice, threadID)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "not found this threads"})
		return
	}

	json.NewEncoder(w).Encode(threadResult)
}
