package main

import (
	"./models"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"log"
	"net/http"
)

var db *sqlx.DB

func main() {
	r := mux.NewRouter()
	connectionString := "dbname=homework user=andrey password=167839 host=localhost port=5432"
	var err error
	db, err = sqlx.Open("postgres", connectionString)
	if err != nil {
		log.Fatal(err)
	}
	r.HandleFunc("/user/{nickname}/create", createUser)
	r.HandleFunc("/user/{nickname}/profile", updateUser)
	http.Handle("/", r)
	http.ListenAndServe(":8080", nil)
}

//TODO: проверка на занятого юзера
func createUser(w http.ResponseWriter, r *http.Request) {
	newUserNickname := mux.Vars(r)["nickname"] //take user nickname
	newUser := &models.User{}                  //form for user data
	newUser.Nickname = newUserNickname
	err := json.NewDecoder(r.Body).Decode(newUser)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Invalid json !"`))
		return
	}
	userInsertState := `insert into users (fullname, email, about, nickname) values ($1, $2, $3, $4);`
	result, err := db.Exec(userInsertState, newUser.Fullname, newUser.Email, newUser.About, newUser.Nickname)
	if id, err := result.LastInsertId(); id < 0 && err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Invalid json !"`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newUser)
}

func updateUser(w http.ResponseWriter, r *http.Request) {
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
	result, err := db.Exec(userUpdateState, newUser.Fullname, newUser.Email, newUser.About, newUser.Nickname)

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
		return
	}

	//отправим измененного юзера обратно
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newUser)
}
