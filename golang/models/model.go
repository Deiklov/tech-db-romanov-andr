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
	Author  string    `json:"author"`
	Created time.Time `json:"created"`
	Forum   string    `json:"forum"`
	Id      int       `json:"id"`
	Message string    `json:"message"`
	Slug    string    `json:"slug"`
	Title   string    `json:"title"`
	Votes   int       `json:"votes"`
}
