package handlers

import (
	"database/sql"
	"github.com/Deiklov/tech-db-romanov-andr/golang/models"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jmoiron/sqlx"
	"github.com/mailru/easyjson"
	"github.com/valyala/fasthttp"
	"net/http"
)

type Handler struct {
	DB   *sqlx.DB
	Conn *pgxpool.Pool
}

func (h *Handler) CreateUser(ctx *fasthttp.RequestCtx) {
	newUserNickname := ctx.UserValue("nickname").(string) //take user nickname
	newUser := &models.User{}                             //form for user data
	newUser.Nickname = newUserNickname
	err := easyjson.Unmarshal(ctx.PostBody(), newUser)
	if err != nil {
		ctx.SetStatusCode(http.StatusInternalServerError)
		ctx.Write([]byte(`{"error": "Invalid json !"`))
		return
	}
	userInsertState := `insert into users (fullname, email, about, nickname) values ($1, $2, $3, $4) returning nickname;`
	result := h.DB.QueryRow(userInsertState, newUser.Fullname, newUser.Email, newUser.About, newUser.Nickname)
	var nickname string
	err = result.Scan(&nickname)

	if err, ok := err.(pgx.PgError); ok {
		switch err.Code {
		//this is conflict code
		case "23505":
			ctx.SetStatusCode(http.StatusConflict)
			items := models.UserSet{}
			userInsertState := `SELECT about,email,fullname,nickname from users where lower(email)=lower($1) or lower(nickname)=lower($2);`
			err := h.DB.Select(&items, userInsertState, newUser.Email, newUser.Nickname)
			if err != nil {
				ctx.SetStatusCode(http.StatusInternalServerError)
				return
			}
			//может вернуть несколько челиков(с одним почта совпала с другим логин)
			data, _ := easyjson.Marshal(items)
			ctx.Write(data)
			return
		default:
			ctx.SetStatusCode(http.StatusInternalServerError)
			ctx.Write([]byte(`{"error": "Invalid json !"`))
			return
		}
	}
	//вернем 409 и существующего юзера
	ctx.SetStatusCode(http.StatusCreated)
	data, _ := easyjson.Marshal(newUser)
	ctx.Write(data)
}

func (h *Handler) UpdateUser(ctx *fasthttp.RequestCtx) {
	newUserNickname := ctx.UserValue("nickname").(string) //take user nickname
	newUser := &models.User{}                             //form for user data
	newUser.Nickname = newUserNickname
	err := easyjson.Unmarshal(ctx.PostBody(), newUser)
	if err != nil {
		ctx.SetStatusCode(500)
		ctx.Write([]byte(`{"error": "Invalid json !"`))
		return
	}

	userUpdateState := `update users set nickname='` + newUserNickname + `' `

	if newUser.Fullname.Valid {
		userUpdateState += ` ,fullname='` + newUser.Fullname.String + `'`
	}
	if newUser.Email.Valid {
		userUpdateState += ` ,email='` + newUser.Email.String + `'`
	}
	if newUser.About.Valid {
		userUpdateState += ` ,about='` + newUser.About.String + `' `
	}
	userUpdateState += ` where lower(nickname)= lower($1) returning *`
	err = h.DB.Get(newUser, userUpdateState, newUser.Nickname)

	//проверка на уникальность email and nickname
	if err == sql.ErrNoRows {
		ctx.SetStatusCode(404)
		data, _ := easyjson.Marshal(models.NotFoundMsg)
		ctx.Write(data)
		return
	}

	if err, ok := err.(pgx.PgError); ok {
		switch err.Code {
		case "23505":
			ctx.SetStatusCode(409)
			data, _ := easyjson.Marshal(models.ConflictMsg)
			ctx.Write(data)
			return
		default:
			ctx.SetStatusCode(500)
			return
		}
	}

	data, _ := easyjson.Marshal(newUser)
	ctx.Write(data)
}
func (h *Handler) GetUser(ctx *fasthttp.RequestCtx) {
	userNickname := ctx.UserValue("nickname").(string) //take user nickname
	user := &models.User{}                             //form for user data
	userQuery := `SELECT about,email,fullname,nickname from users where lower(nickname)=lower($1);`
	err := h.DB.Get(user, userQuery, userNickname)

	switch {
	case err == sql.ErrNoRows:
		ctx.SetStatusCode(404)
		data, _ := easyjson.Marshal(models.NotFoundMsg)
		ctx.Write(data)
		return
	case err != nil:
		ctx.SetStatusCode(500)
		return
	}

	if user.Nickname == "" {
		ctx.SetStatusCode(404)
		data, _ := easyjson.Marshal(models.NotFoundMsg)
		ctx.Write(data)
		return
	}

	data, _ := easyjson.Marshal(user)
	ctx.Write(data)
}
