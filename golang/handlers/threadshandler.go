package handlers

import (
	"database/sql"
	"github.com/Deiklov/tech-db-romanov-andr/golang/models"
	"github.com/jackc/pgx"
	"github.com/mailru/easyjson"
	"github.com/valyala/fasthttp"
	"net/http"
	"strconv"
)

func (h *Handler) ThreadInfo(ctx *fasthttp.RequestCtx) {
	thread := &models.Thread{}
	id, err := h.toID(ctx)
	if err != nil {
		ctx.SetStatusCode(404)
		data, _ := easyjson.Marshal(models.NotFoundMsg)
		ctx.Write(data)
		return
	}

	if err := h.DB.Get(thread, `SELECT * from threads where id=$1`, id); err != nil {
		ctx.SetStatusCode(500)
		return
	}

	data, _ := easyjson.Marshal(thread)
	ctx.Write(data)
}

func (h *Handler) ThreadUpdate(ctx *fasthttp.RequestCtx) {
	id, err := h.toID(ctx)
	if err != nil {
		ctx.SetStatusCode(404)
		data, _ := easyjson.Marshal(models.NotFoundMsg)
		ctx.Write(data)
		return
	}
	threadUPD := &models.ThreadUpdate{}
	threadResult := &models.Thread{}
	err = easyjson.Unmarshal(ctx.PostBody(), threadUPD)
	if err != nil {
		ctx.SetStatusCode(http.StatusInternalServerError)
		ctx.Write([]byte(`{"error": "Invalid json !"`))
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
		data, _ := easyjson.Marshal(threadResult)
		ctx.Write(data)
		return
	}

	queryThread += ` returning *;`
	err = h.DB.Get(threadResult, queryThread)
	if err != nil {
		if threadResult.Id <= 0 {
			ctx.SetStatusCode(404)
			data, _ := easyjson.Marshal(models.NotFoundMsg)
			ctx.Write(data)
			return
		}
		ctx.SetStatusCode(500)
		return
	}
	data, _ := easyjson.Marshal(threadResult)
	ctx.Write(data)
	return
}

func (h *Handler) ThreadVotes(ctx *fasthttp.RequestCtx) {
	voice := &models.Vote{}
	threadResult := &models.Thread{}
	_ = easyjson.Unmarshal(ctx.PostBody(), voice)

	threadID, err := h.toID(ctx)
	if err != nil {
		ctx.SetStatusCode(404)
		data, _ := easyjson.Marshal(models.NotFoundMsg)
		ctx.Write(data)
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
			ctx.SetStatusCode(404)
			data, _ := easyjson.Marshal(models.NotFoundMsg)
			ctx.Write(data)
			return
		}
	}

	err = h.DB.Get(threadResult, `select * from threads where id=$1`, threadID)

	if err == sql.ErrNoRows {
		ctx.SetStatusCode(404)
		data, _ := easyjson.Marshal(models.NotFoundMsg)
		ctx.Write(data)
		return
	}

	data, _ := easyjson.Marshal(threadResult)
	ctx.Write(data)
}
