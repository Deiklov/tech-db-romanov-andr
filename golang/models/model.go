package models

import (
	"gopkg.in/guregu/null.v3"
	"gopkg.in/guregu/null.v3/zero"
	"time"
)

type User struct {
	About    null.String `json:"about,omitempty"`
	Email    null.String `json:"email,omitempty"`
	Fullname null.String `json:"fullname,omitempty"`
	Nickname string      `json:"nickname"`
}

type Forum struct {
	Posts    int    `json:"posts"`
	Slug     string `json:"slug"`
	Threads  int    `json:"threads"`
	Title    string `json:"title"`
	UserNick string `json:"user" db:"user"`
}

type Thread struct {
	Author  string      `json:"author" db:"author"`
	Created time.Time   `json:"created" db:"created"`
	Forum   string      `json:"forum" db:"forum"`
	Id      int         `json:"id" db:"id"`
	Message string      `json:"message" db:"message"`
	Slug    null.String `json:"slug,omitempty" db:"slug"`
	Title   string      `json:"title" db:"title"`
	Votes   int         `json:"votes" db:"votes"`
}

type ThreadParams struct {
	Limit int       `schema:"limit"`
	Since time.Time `schema:"since"`
	Desc  bool      `schema:"desc"`
}

type ForumUserParams struct {
	Limit int    `schema:"limit"`
	Since string `schema:"since"`
	Desc  bool   `schema:"desc"`
}
type ThreadUpdate struct {
	Message string `json:"message,omitempty"`
	Title   string `json:"title,omitempty"`
}

type Vote struct {
	Nickname string `json:"nickname"`
	Voice    int    `json:"voice"`
}

type Info struct {
	Forum  uint64 `json:"forum"`
	Post   uint64 `json:"post"`
	Thread uint64 `json:"thread"`
	User   uint64 `json:"user"`
}

type Post struct {
	Author   string    `json:"author" db:"author"`
	Created  time.Time `json:"created" db:"created"`
	Forum    string    `json:"forum" db:"forum"`
	Id       int       `json:"id" db:"id"`
	IsEdited bool      `json:"isEdited" db:"isedited"`
	Message  string    `json:"message" db:"message"`
	Parent   zero.Int  `json:"parent" db:"parent"`
	Thread   int       `json:"thread" db:"thread"`
	Path     []int     `json:"-" db:"path"`
}

type PostUpdate struct {
	Message string `json:"message" db:"message"`
}

type PostParams struct {
	Limit int
	Since int
	Sort  string
	Desc  bool
}

type VotesInfo struct {
	Votes    int    `db:"votes"`
	ThreadID int    `db:"thread_id"`
	Nickname string `db:"nickname"`
}

//easyjson:json
type PostSet []Post

//easyjson:json
type ThreadSet []Thread

//easyjson:json
type UserSet []User

//easyjson:json
type NotFoundMes map[string]string

var NotFoundMsg = NotFoundMes{"message": "Not found"}
var ConflictMsg = NotFoundMes{"message": "Conflict"}
