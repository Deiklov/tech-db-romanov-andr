package handlers

import (
	"../models"
	"encoding/json"
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
		queryThread = `SELECT * from threads where slug=$1`
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

	if thread.Slug == "" {
		w.WriteHeader(http.StatusNotFound)
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
		queryThread += `slug='` + slug + `'`
	} else {
		queryThread += `id=` + strconv.Itoa(id)
	}
	queryThread += ` returning *;`
	err = h.DB.Get(threadResult, queryThread)
	if err != nil {
		if threadResult.Slug == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(threadResult)
}

func (h *Handler) ThreadVotes(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug_or_id"]
	voice := &models.Vote{}
	threadResult := &models.Thread{}
	json.NewDecoder(r.Body).Decode(voice)
	queryThread := ""
	id, err := strconv.Atoi(slug)
	queryThread = `update threads set votes=votes+` + strconv.Itoa(voice.Voice) + ` where `
	if err != nil {
		queryThread += `slug='` + slug + `'`
	} else {
		queryThread += `id=` + strconv.Itoa(id)
	}
	queryThread += ` returning *;`
	err = h.DB.Get(threadResult, queryThread)
	if err != nil {
		if threadResult.Slug == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(threadResult)
}
