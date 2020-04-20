package handlers

import (
	"database/sql"
	"github.com/Deiklov/tech-db-romanov-andr/golang/models"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx"
	"github.com/jmoiron/sqlx"
	"github.com/mailru/easyjson"
	"net/http"
)

type Handler struct {
	DB *sqlx.DB
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	newUserNickname := mux.Vars(r)["nickname"] //take user nickname
	newUser := &models.User{}                  //form for user data
	newUser.Nickname = newUserNickname
	err := easyjson.UnmarshalFromReader(r.Body, newUser)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Invalid json !"`))
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
			w.WriteHeader(http.StatusConflict)
			items := models.UserSet{}
			userInsertState := `SELECT about,email,fullname,nickname from users where lower(email)=lower($1) or lower(nickname)=lower($2);`
			err := h.DB.Select(&items, userInsertState, newUser.Email, newUser.Nickname)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			//может вернуть несколько челиков(с одним почта совпала с другим логин)
			easyjson.MarshalToHTTPResponseWriter(items, w)
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Some error with data querys!"))
			return
		}
	}
	//вернем 409 и существующего юзера
	w.WriteHeader(http.StatusCreated)
	easyjson.MarshalToHTTPResponseWriter(newUser, w)
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	newUserNickname := mux.Vars(r)["nickname"] //take user nickname
	newUser := &models.User{}                  //form for user data
	newUser.Nickname = newUserNickname
	err := easyjson.UnmarshalFromReader(r.Body, newUser)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Invalid json !"`))
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
		w.WriteHeader(http.StatusNotFound)
		easyjson.MarshalToHTTPResponseWriter(models.NotFoundMsg, w)
		return
	}

	if err, ok := err.(pgx.PgError); ok {
		switch err.Code {
		case "23505":
			w.WriteHeader(http.StatusConflict)
			easyjson.MarshalToHTTPResponseWriter(models.ConflictMsg, w)
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Some error with data querys!"))
			return
		}
	}

	easyjson.MarshalToHTTPResponseWriter(newUser, w)
}
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	userNickname := mux.Vars(r)["nickname"] //take user nickname
	user := &models.User{}                  //form for user data
	userQuery := `SELECT about,email,fullname,nickname from users where lower(nickname)=lower($1);`
	err := h.DB.Get(user, userQuery, userNickname)

	switch {
	case err == sql.ErrNoRows:
		w.WriteHeader(http.StatusNotFound)
		easyjson.MarshalToHTTPResponseWriter(models.NotFoundMsg, w)
		return
	case err != nil:
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if user.Nickname == "" {
		w.WriteHeader(http.StatusNotFound)
		easyjson.MarshalToHTTPResponseWriter(models.NotFoundMsg, w)
		return
	}

	easyjson.MarshalToHTTPResponseWriter(user, w)
}
