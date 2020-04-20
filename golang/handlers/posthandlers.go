package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/Deiklov/tech-db-romanov-andr/golang/models"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/jackc/pgx"
	"github.com/mailru/easyjson"
	"github.com/pkg/errors"
	"gopkg.in/guregu/null.v3"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (h *Handler) CreatePost(w http.ResponseWriter, r *http.Request) {
	postResult := models.PostSet{}
	if err := easyjson.UnmarshalFromReader(r.Body, &postResult); err != nil {
		http.Error(w, "err in easyjson", 500)
		return
	}
	threadId, err := h.toID(r)

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		easyjson.MarshalToHTTPResponseWriter(models.NotFoundMsg, w)
		return
	}

	var forumSlug string
	if err := h.DB.Get(&forumSlug, `select forums.slug from forums
		inner join threads t on forums.slug = t.forum
	where t.id=$1 limit 1`, threadId); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	currTime := time.Now().UTC()
	err = h.bulkInsert(postResult, forumSlug, threadId, currTime)

	if err != nil {
		if err, ok := err.(pgx.PgError); ok {
			switch err.Message {
			case "invalid parent id":
				w.WriteHeader(http.StatusConflict)
				easyjson.MarshalToHTTPResponseWriter(models.ConflictMsg, w)
				return
			case "not found author":
				w.WriteHeader(http.StatusNotFound)
				easyjson.MarshalToHTTPResponseWriter(models.NotFoundMsg, w)
				return
			default:
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}

	retPosts := models.PostSet{}

	err = h.DB.
		Select(&retPosts, `select * from posts where created=$1 order by id`, currTime)

	if err == sql.ErrNoRows {
		if _, _, err := easyjson.MarshalToHTTPResponseWriter(retPosts, w); err != nil {
			http.Error(w, "easyjson err", 500)
			return
		}
		w.WriteHeader(http.StatusCreated)
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusCreated)

	if _, _, err := easyjson.MarshalToHTTPResponseWriter(retPosts, w); err != nil {
		http.Error(w, "easyjson err", 500)
		return
	}

}
func (h *Handler) GetPost(w http.ResponseWriter, r *http.Request) {
	stringID := mux.Vars(r)["id"]
	postID, err := strconv.Atoi(stringID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	paramsInString := r.URL.Query().Get("related")
	related := strings.Split(paramsInString, ",")
	retPost := models.Post{}
	//нету параметров
	if err := h.DB.Get(&retPost, `select * from posts where id=$1`, postID); err != nil {
		switch {
		case err == sql.ErrNoRows:
			w.WriteHeader(http.StatusNotFound)
			easyjson.MarshalToHTTPResponseWriter(models.NotFoundMsg, w)
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	thread := models.Thread{}
	forum := models.Forum{}
	author := models.User{}

	for _, v := range related {
		switch v {
		case "user":
			if err := h.DB.Get(&author, `select * from users where lower(nickname)=lower($1)`, retPost.Author); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		case "forum":
			if err := h.DB.Get(&forum, `select * from forums where lower(slug)=lower($1)`, retPost.Forum); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		case "thread":
			if err := h.DB.Get(&thread, `select * from threads where id=$1`, retPost.Thread); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}

	res := make(map[string]interface{})

	res["post"] = retPost

	if thread.Title != "" {
		res["thread"] = thread
	}
	if author.Email.String != "" {
		res["author"] = author
	}
	if forum.Slug != "" {
		res["forum"] = forum
	}

	json.NewEncoder(w).Encode(&res)

}
func (h *Handler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	stringID := mux.Vars(r)["id"]
	postID, err := strconv.Atoi(stringID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	retPost := models.Post{}
	newPostData := models.PostUpdate{}

	err = easyjson.UnmarshalFromReader(r.Body, &newPostData)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Invalid json !"`))
		return
	}

	if err := h.DB.Get(&retPost, `select * from posts where id=$1`, postID); err != nil {
		switch {
		case err == sql.ErrNoRows:
			w.WriteHeader(http.StatusNotFound)
			easyjson.MarshalToHTTPResponseWriter(models.NotFoundMsg, w)
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	updatedPost := models.Post{}
	//то же самое сообщение
	if retPost.Message == newPostData.Message || newPostData.Message == "" {
		easyjson.MarshalToHTTPResponseWriter(retPost, w)
		return
	}

	err = h.DB.Get(&updatedPost, `update posts set message=$1, isedited=true where id=$2 returning *`, newPostData.Message, postID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	easyjson.MarshalToHTTPResponseWriter(updatedPost, w)
}

func (h *Handler) GetAllPosts(w http.ResponseWriter, r *http.Request) {
	//взяли id треда
	id, err := h.toID(r)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		easyjson.MarshalToHTTPResponseWriter(models.NotFoundMsg, w)
		return
	}

	postList := models.PostSet{}

	params := &models.PostParams{}
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	if err := decoder.Decode(params, r.URL.Query()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	query := `select * from posts where thread=$1 `

	if params.Since > 0 {
		if params.Desc {
			query += ` and id<` + strconv.Itoa(params.Since)
		} else {
			query += ` and id>` + strconv.Itoa(params.Since)
		}
	}

	if params.Desc {
		query += ` order by created desc, id desc `
	} else {
		query += ` order by created, id `
	}

	if params.Limit > 0 {
		query += ` limit ` + strconv.Itoa(params.Limit)
	}

	switch params.Sort {
	case "tree":
		rootPostList := models.PostSet{}
		resultPostList := models.PostSet{}
		//shadow var query
		query := `select * from posts where thread=$1 and parent is null order by id `

		if params.Desc {
			query += ` desc`
		}

		err = h.DB.Select(&rootPostList, query, id)

		for _, v := range rootPostList {
			if !params.Desc {
				resultPostList = append(resultPostList, v)
			}
			resultPostList = append(resultPostList, h.getPosts(v.Id, id, params.Desc)...)
			if params.Desc {
				resultPostList = append(resultPostList, v)
			}
		}

		res := models.PostSet{}

		if params.Since > 0 {
			for i, v := range resultPostList {
				if v.Id == params.Since {
					res = append(res, resultPostList[i+1:len(resultPostList)]...)
					break
				}
			}
			resultPostList = res
		}

		if params.Limit > 0 {
			if len(resultPostList) > params.Limit {
				resultPostList = resultPostList[:params.Limit]
			}
		}
		if _, _, err := easyjson.MarshalToHTTPResponseWriter(resultPostList, w); err != nil {
			http.Error(w, "easy", 500)
			return
		}
		return
	case "parent_tree":
		rootPostList := models.PostSet{}
		resultPostList := models.PostSet{}
		//shadow var query
		query := `select * from posts where thread=$1 and parent is null order by id `

		if params.Desc {
			query += ` desc`
		}

		err = h.DB.Select(&rootPostList, query, id)
		//todo не будет работать при since
		if params.Limit > 0 {
			if params.Since == 0 {
				if params.Limit < len(rootPostList) {
					rootPostList = rootPostList[:params.Limit]
				}
			}
		}

		for _, v := range rootPostList {
			resultPostList = append(resultPostList, v)
			resultPostList = append(resultPostList, h.getPosts(v.Id, id, false)...)

		}

		res := models.PostSet{}

		if params.Since > 0 {
			for i, v := range resultPostList {
				if v.Id == params.Since {
					res = append(res, resultPostList[i+1:len(resultPostList)]...)
					break
				}
			}
			resultPostList = res
		}

		if _, _, err := easyjson.MarshalToHTTPResponseWriter(resultPostList, w); err != nil {
			http.Error(w, "easy", 500)
			return
		}
		return
	default:
	}
	err = h.DB.Select(&postList, query, id)

	if _, _, err := easyjson.MarshalToHTTPResponseWriter(postList, w); err != nil {
		http.Error(w, "easy", 500)
		return
	}
	return

}
func (h *Handler) toID(r *http.Request) (int, error) {
	slugOrID := mux.Vars(r)["slug_or_id"]
	threadId := -1
	id, err := strconv.Atoi(slugOrID)
	if err != nil {
		slug := slugOrID
		if err := h.DB.Get(&threadId, "select id from threads where lower(slug)=lower($1) limit 1", slug); err != nil {
			return -1, errors.New("not found")
		}
	} else {
		threadId = id
	}

	err = h.DB.Get(&threadId, "select id from threads where id=$1", threadId)
	if err == sql.ErrNoRows {
		return -1, errors.New("not found")
	}
	return threadId, nil
}
func (h *Handler) getPosts(parentID int, threadID int, desc bool) models.PostSet {
	postData := models.PostSet{}
	query := `select * from posts where thread=$1 and parent=$2 order by id`
	if desc {
		query += ` desc`
	}
	err := h.DB.Select(&postData, query, threadID, parentID)
	if err != nil {
		return nil
	}

	resultPostData := models.PostSet{}

	if len(postData) == 0 {
		return postData
	} else {
		for _, v := range postData {

			if !desc {
				resultPostData = append(resultPostData, v)
			}
			resultPostData = append(resultPostData, h.getPosts(v.Id, threadID, desc)...)

			if desc {
				resultPostData = append(resultPostData, v)
			}

		}
		return resultPostData
	}
}
func (h *Handler) bulkInsert(unsavedRows models.PostSet, slug string, threadID int, created time.Time) error {
	valueStrings := make([]string, 0, len(unsavedRows))
	if cap(valueStrings) == 0 {
		return nil
	}
	valueArgs := make([]interface{}, 0, len(unsavedRows)*6)
	i := 0
	for _, post := range unsavedRows {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d)", i*6+1, i*6+2, i*6+3, i*6+4, i*6+5, i*6+6))
		valueArgs = append(valueArgs, post.Author)
		valueArgs = append(valueArgs, created)
		valueArgs = append(valueArgs, slug)
		valueArgs = append(valueArgs, post.Message)
		if post.Parent.Int64 != 0 {
			valueArgs = append(valueArgs, post.Parent)
		} else {
			valueArgs = append(valueArgs, null.String{})
		}
		valueArgs = append(valueArgs, threadID)
		i++
	}
	stmt := fmt.Sprintf("insert into posts(author,created,forum,message,parent,thread) VALUES %s", strings.Join(valueStrings, ","))
	_, err := h.DB.Exec(stmt, valueArgs...)
	return err
}
