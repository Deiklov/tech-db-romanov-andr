package handlers

import (
	"../models"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)

func (h *Handler) ThreadInfo(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug_or_id"]
	thread := &models.Thread{}
	queryThread := ""
	id, err := strconv.Atoi(slug)
	if err != nil {
		queryThread = `SELECT * from threads where lower(slug)=lower($1)`
		if err := h.DB.Get(thread, queryThread, slug); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else {
		queryThread = `SELECT * from threads where id=$1`
		if err := h.DB.Get(thread, queryThread, id); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	if thread.Id < 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "not found this threads"})
		return
	}
	json.NewEncoder(w).Encode(thread)
}

func (h *Handler) ThreadUpdate(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug_or_id"]
	thread := &models.ThreadUpdate{}
	threadResult := &models.Thread{}
	json.NewDecoder(r.Body).Decode(&thread)
	queryThread := ""
	id, err := strconv.Atoi(slug)
	queryThread = `update threads set`
	if thread.Message != "" {
		queryThread += ` message='` + thread.Message + `', `
	}
	if thread.Title != "" {
		queryThread += ` title='` + thread.Title + `' where `
	}
	if err != nil {
		queryThread += `lower(slug)=lower('` + slug + `')`
	} else {
		queryThread += `id=` + strconv.Itoa(id)
	}
	queryThread += ` returning *;`
	err = h.DB.Get(threadResult, queryThread)
	if err != nil {
		if threadResult.Id < 0 {
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
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	err = h.DB.Get(threadResult,
		`update threads set votes=votes+($1) where id=$2 returning *`, voice.Voice, threadID)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "not found this threads"})
		return
	}

	fmt.Println(threadResult.Votes)
	json.NewEncoder(w).Encode(threadResult)
}
