package handlers

import (
	"github.com/Deiklov/tech-db-romanov-andr/golang/models"
	"github.com/mailru/easyjson"
	"github.com/valyala/fasthttp"
)

func (h *Handler) ServiceInfo(ctx *fasthttp.RequestCtx) {
	serviceInfo := &models.Info{}
	forumsQuery := `select count(*) forum from forums`
	usersQuery := `select count(*) "user" from users`
	threadsQuery := `select count(*) thread from threads`
	postsQuery := `select count(*) post from posts`
	if err := h.DB.Get(serviceInfo, forumsQuery); err != nil {
		ctx.SetStatusCode(500)
		return
	}
	if err := h.DB.Get(serviceInfo, usersQuery); err != nil {
		ctx.SetStatusCode(500)
		return
	}
	if err := h.DB.Get(serviceInfo, postsQuery); err != nil {
		ctx.SetStatusCode(500)
		return
	}
	if err := h.DB.Get(serviceInfo, threadsQuery); err != nil {
		ctx.SetStatusCode(500)
		return
	}
	data, _ := easyjson.Marshal(serviceInfo)
	ctx.Write(data)
}

func (h *Handler) ServiceClear(ctx *fasthttp.RequestCtx) {
	stmtdelete := `TRUNCATE users,forums,posts,threads,user_forum,votes_info restart identity cascade`
	if _, err := h.DB.Exec(stmtdelete); err != nil {
		ctx.SetStatusCode(500)
		return
	}
}
