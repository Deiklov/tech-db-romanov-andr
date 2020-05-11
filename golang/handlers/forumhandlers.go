package handlers

import (
	"database/sql"
	"github.com/Deiklov/tech-db-romanov-andr/golang/models"
	"github.com/gorilla/schema"
	"github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/mailru/easyjson"
	"github.com/valyala/fasthttp"
	"net/http"
	"strconv"
	"time"
)

func (h *Handler) CreateForum(ctx *fasthttp.RequestCtx) {
	newForum := &models.Forum{}
	_ = easyjson.Unmarshal(ctx.PostBody(), newForum)

	//чекаем есть ли юзер
	var nickname string
	err := h.DB.QueryRow(`select nickname from users where lower(nickname)=lower($1);`, newForum.UserNick).Scan(&nickname)
	//если нет юзера, то кидаем 404
	if err == sql.ErrNoRows {
		ctx.SetStatusCode(404)
		data, _ := easyjson.Marshal(models.NotFoundMsg)
		ctx.Write(data)
		return
	}

	newForum.UserNick = nickname
	backSlug := ""

	queryForum := `insert into forums (slug, title,"user") values($1,$2,$3) returning slug;`
	err = h.DB.Get(&backSlug, queryForum, newForum.Slug, newForum.Title, newForum.UserNick)

	if err, ok := err.(pgx.PgError); ok {
		switch err.Code {
		//this is conflict code
		case "23505":
			ctx.SetStatusCode(409)
			oldForum := &models.Forum{}

			userInsertState := `SELECT * from forums where lower(slug)=lower($1);`
			if err := h.DB.Get(oldForum, userInsertState, newForum.Slug); err != nil {
				ctx.SetStatusCode(http.StatusInternalServerError)
				return
			}

			data, _ := easyjson.Marshal(oldForum)
			ctx.Write(data)
			return
		default:
			ctx.SetStatusCode(http.StatusInternalServerError)
			return
		}
	}

	ctx.SetStatusCode(http.StatusCreated)
	data, _ := easyjson.Marshal(newForum)
	ctx.Write(data)
}

func (h *Handler) ForumDetails(ctx *fasthttp.RequestCtx) {
	slug := ctx.UserValue("slug").(string)
	foundForum := &models.Forum{}
	err := h.DB.Get(foundForum, `select * from forums where lower(slug)=lower($1)`, slug)

	if err == sql.ErrNoRows {
		ctx.SetStatusCode(404)
		data, _ := easyjson.Marshal(models.NotFoundMsg)
		ctx.Write(data)
		return
	}

	data, _ := easyjson.Marshal(foundForum)
	ctx.Write(data)
}

func (h *Handler) NewThread(ctx *fasthttp.RequestCtx) {
	slug := ctx.UserValue("slug").(string)
	forumDB := models.Forum{}
	authorDB := models.User{}

	newThrd := &models.Thread{}
	_ = easyjson.Unmarshal(ctx.PostBody(), newThrd)

	err := h.DB.Get(&forumDB, `select * from forums where lower(slug)=lower($1)`, slug)
	if err == sql.ErrNoRows {
		ctx.SetStatusCode(404)
		data, _ := easyjson.Marshal(models.NotFoundMsg)
		ctx.Write(data)
		return
	}

	err = h.DB.Get(&authorDB, `select * from users where lower(nickname)=lower($1)`, newThrd.Author)

	if err == sql.ErrNoRows {
		ctx.SetStatusCode(404)
		data, _ := easyjson.Marshal(models.NotFoundMsg)
		ctx.Write(data)
		return
	}

	newThrd.Author = authorDB.Nickname
	newThrd.Forum = forumDB.Slug
	newThrd.Created = newThrd.Created.UTC()
	queryThreads := `insert into threads (author, forum, message, title,created) values($1,$2,$3,$4,$5) returning *;`
	if newThrd.Slug.Valid {
		queryThreads = `insert into threads (author, forum, message, title,created,slug) values($1,$2,$3,$4,$5,'` +
			newThrd.Slug.String + `') returning *;`
	}
	err = h.DB.Get(newThrd, queryThreads, newThrd.Author, newThrd.Forum, newThrd.Message, newThrd.Title, newThrd.Created)

	if err != nil {
		if err, ok := err.(pgx.PgError); ok {
			switch err.Code {
			//не вставит если нет юзера или форума
			case "23503":
				ctx.SetStatusCode(404)
				data, _ := easyjson.Marshal(models.NotFoundMsg)
				ctx.Write(data)
				return
			case "23505":
				ctx.SetStatusCode(409)
				exsistThread := models.Thread{}
				h.DB.Get(&exsistThread, `select * from threads where lower(slug)=lower($1)`, newThrd.Slug.String)
				data, _ := easyjson.Marshal(exsistThread)
				ctx.Write(data)
				return
			default:
				ctx.SetStatusCode(500)
				return
			}
		}
	}

	ctx.SetStatusCode(201)
	data, _ := easyjson.Marshal(newThrd)
	ctx.Write(data)
}

func (h *Handler) AllThreadsFromForum(ctx *fasthttp.RequestCtx) {
	params := &models.ThreadParams{}
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	argsList := make(map[string][]string)

	ctx.QueryArgs().VisitAll(func(key, value []byte) {
		argsList[string(key)] = []string{string(value)}
	})

	if err := decoder.Decode(params, argsList); err != nil {
		ctx.SetStatusCode(500)
		return
	}

	slug := ctx.UserValue("slug").(string)

	items := models.ThreadSet{}
	params.Since = params.Since.UTC()

	forumSlugFromDB := ""
	err := h.DB.Get(&forumSlugFromDB, `select slug from forums where lower(slug)=lower($1)`, slug)
	if err == sql.ErrNoRows {
		ctx.SetStatusCode(404)
		data, _ := easyjson.Marshal(models.NotFoundMsg)
		ctx.Write(data)
		return
	}

	threadsQuery := `SELECT author,created,forum,id,message,slug,title,votes
from threads where lower(forum)=lower($1) `

	if params.Desc {
		zeroTime := time.Time{}
		//если не указали время то при деск created нужен только для соответствия call(2 args)
		if params.Since == zeroTime {
			threadsQuery += ` and created >=$2 order by created desc `
		} else {
			threadsQuery += ` and created <=$2 order by created desc `
		}

	} else {
		threadsQuery += ` and created >=$2 order by created `
	}

	if params.Limit > 0 {
		threadsQuery += `limit ` + strconv.Itoa(params.Limit)
	}

	err = h.DB.Select(&items, threadsQuery, slug, params.Since)

	if err == sql.ErrNoRows {
		data, _ := easyjson.Marshal(items)
		ctx.Write(data)
	}

	if err != nil {
		ctx.SetStatusCode(500)
		ctx.Write([]byte(err.Error()))
		return
	}

	data, _ := easyjson.Marshal(items)
	ctx.Write(data)
}

func (h *Handler) AllUsersForum(ctx *fasthttp.RequestCtx) {
	params := &models.ForumUserParams{}
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	argsList := make(map[string][]string)

	ctx.QueryArgs().VisitAll(func(key, value []byte) {
		argsList[string(key)] = []string{string(value)}
	})

	if err := decoder.Decode(params, argsList); err != nil {
		ctx.SetStatusCode(500)
		return
	}

	slug := ctx.UserValue("slug").(string)

	forumSlug := ""
	err := h.DB.Get(&forumSlug, `select slug from forums where lower(slug)=lower($1)`, slug)
	if err == sql.ErrNoRows {
		ctx.SetStatusCode(404)
		data, _ := easyjson.Marshal(models.NotFoundMsg)
		ctx.Write(data)
		return
	}

	users := models.UserSet{}

	switch {
	case params.Desc == false && params.Limit == 0 && params.Since == "":
		err = h.DB.Select(&users, `select about, email, fullname, users.nickname
			from user_forum
         join users on users.nickname = user_forum.nickname
			where forum = $1 order by lower(users.nickname)`, forumSlug)

	case params.Desc == false && params.Limit == 0 && params.Since != "":
		err = h.DB.Select(&users, `select about, email, fullname, users.nickname
			from user_forum
         join users on users.nickname = user_forum.nickname
			where forum = $1 and lower(user_forum.nickname)>lower($2) order by lower(users.nickname)`, forumSlug, params.Since)

	case params.Desc == false && params.Limit > 0 && params.Since == "":
		err = h.DB.Select(&users, `select about, email, fullname, users.nickname
			from user_forum
         join users on users.nickname = user_forum.nickname
			where forum = $1 order by lower(users.nickname) limit $2`, forumSlug, params.Limit)

	case params.Desc == false && params.Limit > 0 && params.Since != "":
		err = h.DB.Select(&users, `select about, email, fullname, users.nickname
			from user_forum
         join users on users.nickname = user_forum.nickname
			where forum = $1 and lower(user_forum.nickname)>lower($2) 
			order by lower(users.nickname) limit $3`, forumSlug, params.Since, params.Limit)

	case params.Desc == true && params.Limit == 0 && params.Since == "":
		err = h.DB.Select(&users, `select about, email, fullname, users.nickname
			from user_forum
         join users on users.nickname = user_forum.nickname
			where forum = $1 order by lower(users.nickname) desc`, forumSlug)

	case params.Desc == true && params.Limit == 0 && params.Since != "":
		err = h.DB.Select(&users, `select about, email, fullname, users.nickname
			from user_forum
         join users on users.nickname = user_forum.nickname
			where forum = $1 and lower(user_forum.nickname)<lower($2) order by lower(users.nickname) desc`, forumSlug, params.Since)

	case params.Desc == true && params.Limit > 0 && params.Since == "":
		err = h.DB.Select(&users, `select about, email, fullname, users.nickname
			from user_forum
         join users on users.nickname = user_forum.nickname
			where forum = $1 order by lower(users.nickname) desc limit $2`, forumSlug, params.Limit)

	case params.Desc == true && params.Limit > 0 && params.Since != "":
		err = h.DB.Select(&users, `select about, email, fullname, users.nickname
			from user_forum
         join users on users.nickname = user_forum.nickname
			where forum = $1 and lower(user_forum.nickname)<lower($2) 
			order by lower(users.nickname) desc limit $3`, forumSlug, params.Since, params.Limit)
	}

	data, _ := easyjson.Marshal(users)
	ctx.Write(data)
}
