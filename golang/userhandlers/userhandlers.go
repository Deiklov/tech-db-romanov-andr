package userhandlers

import (
	"../models"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"net/http"
)

type UserHandler struct {
	DB *sqlx.DB
}

//TODO: проверка на занятого юзера
func (handler *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	newUserNickname := mux.Vars(r)["nickname"] //take user nickname
	newUser := &models.User{}                  //form for user data
	newUser.Nickname = newUserNickname
	err := json.NewDecoder(r.Body).Decode(newUser)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Invalid json !"`))
		return
	}
	userInsertState := `insert into users (fullname, email, about, nickname) values ($1, $2, $3, $4) returning nickname;`
	result := handler.DB.QueryRow(userInsertState, newUser.Fullname, newUser.Email, newUser.About, newUser.Nickname)
	var nickname string
	err = result.Scan(&nickname)
	if err, ok := err.(*pq.Error); ok {
		switch err.Code {
		//this is conflict code
		case "23505":
			w.WriteHeader(http.StatusConflict)
			items := []*models.User{}
			userInsertState := `SELECT about,email,fullname,nickname from users where email=$1 or nickname=$2;`
			rows, err := handler.DB.Query(userInsertState, newUser.Email, newUser.Nickname)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("error with select already exsist user"))
				return
			}
			defer rows.Close()
			for rows.Next() {
				oldUser := &models.User{}
				err := rows.Scan(&oldUser.About, &oldUser.Email, &oldUser.Fullname, &oldUser.Nickname)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				items = append(items, oldUser)
			}
			//может вернуть несколько челиков(с одним почта совпала с другим логин)
			json.NewEncoder(w).Encode(items)
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Some error with data querys!"))
			return
		}
	}
	//вернем 409 и существующего юзера
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newUser)
}

func (handler *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	newUserNickname := mux.Vars(r)["nickname"] //take user nickname
	newUser := &models.User{}                  //form for user data
	newUser.Nickname = newUserNickname
	err := json.NewDecoder(r.Body).Decode(newUser)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Invalid json !"`))
		return
	}
	userUpdateState := `UPDATE users SET  fullname= $1, email = $2, about = $3 WHERE nickname= $4;`
	result, err := handler.DB.Exec(userUpdateState, newUser.Fullname, newUser.Email, newUser.About, newUser.Nickname)

	//проверка на уникальность email and nickname
	if err, ok := err.(*pq.Error); ok {
		switch err.Code {
		case "23505":
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"message": "Already exsist user with same nickname or email!"})
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Some error with data querys!"))
			return
		}
	}

	//проверка на существование юзера
	if rowsAffected, _ := result.RowsAffected(); rowsAffected < 1 && err == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Not found user with same nickname!"})
		return
	}

	//отправим измененного юзера обратно
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newUser)
}
func (handler *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	userNickname := mux.Vars(r)["nickname"] //take user nickname
	user := &models.User{}                  //form for user data
	userQuery := `SELECT about,email,fullname,nickname from users where nickname=$1;`
	row := handler.DB.QueryRow(userQuery, userNickname)
	err := row.Scan(&user.About, &user.Email, &user.Fullname, &user.Nickname)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error with scan"))
		return
	}
	if user.Nickname == "" {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Not found user with same nickname!"})
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
	return
}
