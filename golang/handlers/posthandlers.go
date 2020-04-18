package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/Deiklov/tech-db-romanov-andr/golang/models"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (h *Handler) CreatePost(w http.ResponseWriter, r *http.Request) {
	postResult := []*models.Post{}
	json.NewDecoder(r.Body).Decode(&postResult)
	threadId, err := h.toID(r)

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Can't find thread"})
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
		if err, ok := err.(*pq.Error); ok {
			switch err.Message {
			case "invalid parent id":
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]string{"message": "parent doesn't exsist in this thread"})
				return
			case "not found author":
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]string{"message": "author doesn't exsist"})
				return
			default:
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}

	retPosts := []*models.Post{}
	//todo maybe insert in other order
	err = h.DB.
		Select(&retPosts, `select * from posts where created=$1 order by id`, currTime)

	if err == sql.ErrNoRows {
		json.NewEncoder(w).Encode(retPosts)
		w.WriteHeader(http.StatusCreated)
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(retPosts)

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
			json.NewEncoder(w).Encode(map[string]string{"message": "Can't find user with that nickname"})
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

	if err := json.NewDecoder(r.Body).Decode(&newPostData); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := h.DB.Get(&retPost, `select * from posts where id=$1`, postID); err != nil {
		switch {
		case err == sql.ErrNoRows:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"message": "Can't find user with that nickname"})
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	updatedPost := models.Post{}
	//то же самое сообщение
	if retPost.Message == newPostData.Message || newPostData.Message == "" {
		json.NewEncoder(w).Encode(retPost)
		return
	}

	err = h.DB.Get(&updatedPost, `update posts set message=$1, isedited=true where id=$2 returning *`, newPostData.Message, postID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(&updatedPost)
}

func (h *Handler) GetAllPosts(w http.ResponseWriter, r *http.Request) {
	//взяли id треда
	id, err := h.toID(r)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Can't find this thread"})
		return
	}

	postList := []models.Post{}

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
		rootPostList := []models.Post{}
		resultPostList := []models.Post{}
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

		res := []models.Post{}

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

		json.NewEncoder(w).Encode(resultPostList)
		return
	case "parent_tree":
		rootPostList := []models.Post{}
		resultPostList := []models.Post{}
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

		res := []models.Post{}

		if params.Since > 0 {
			for i, v := range resultPostList {
				if v.Id == params.Since {
					res = append(res, resultPostList[i+1:len(resultPostList)]...)
					break
				}
			}
			resultPostList = res
		}

		json.NewEncoder(w).Encode(resultPostList)
		return
	default:
	}
	err = h.DB.Select(&postList, query, id)

	json.NewEncoder(w).Encode(postList)
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
func (h *Handler) getPosts(parentID int, threadID int, desc bool) []models.Post {
	postData := []models.Post{}
	query := `select * from posts where thread=$1 and parent=$2 order by id`
	if desc {
		query += ` desc`
	}
	err := h.DB.Select(&postData, query, threadID, parentID)
	if err != nil {
		return nil
	}

	resultPostData := []models.Post{}

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
func (h *Handler) bulkInsert(unsavedRows []*models.Post, slug string, threadID int, created time.Time) error {
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
			valueArgs = append(valueArgs, sql.NullString{})
		}
		valueArgs = append(valueArgs, threadID)
		i++
	}
	stmt := fmt.Sprintf("insert into posts(author,created,forum,message,parent,thread) VALUES %s", strings.Join(valueStrings, ","))
	_, err := h.DB.Exec(stmt, valueArgs...)
	return err
}
