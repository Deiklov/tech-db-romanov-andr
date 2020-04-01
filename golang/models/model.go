package models

import "time"

type User struct {
	About    string `json:"about"`
	Email    string `json:"email"`
	Fullname string `json:"fullname"`
	Nickname string `json:"nickname"`
}

type Forum struct {
	Posts    int    `json:"posts"`
	Slug     string `json:"slug"`
	Threads  int    `json:"threads"`
	Title    string `json:"title"`
	UserNick string `json:"user"`
}

type Thread struct {
	Author  string    `json:"author" db:"author"`
	Created time.Time `json:"created" db:"created"`
	Forum   string    `json:"forum" db:"forum"`
	Id      int       `json:"id" db:"id"`
	Message string    `json:"message" db:"message"`
	Slug    string    `json:"slug" db:"slug"`
	Title   string    `json:"title" db:"title"`
	Votes   int       `json:"votes" db:"votes"`
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
