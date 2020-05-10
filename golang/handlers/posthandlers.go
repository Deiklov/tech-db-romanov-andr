package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/Deiklov/tech-db-romanov-andr/golang/models"
	"github.com/gorilla/schema"
	"github.com/jackc/pgconn"
	pgx4 "github.com/jackc/pgx/v4"
	"github.com/lib/pq"
	"github.com/mailru/easyjson"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"gopkg.in/guregu/null.v3"
	"log"
	"strconv"
	"strings"
	"time"
)

func (h *Handler) CreatePost(ctx *fasthttp.RequestCtx) {
	postResult := models.PostSet{}
	easyjson.Unmarshal(ctx.PostBody(), &postResult)
	threadId, err := h.toID(ctx)

	if err != nil {
		ctx.SetStatusCode(404)
		data, _ := easyjson.Marshal(models.NotFoundMsg)
		ctx.Write(data)
		return
	}

	var forumSlug string
	if err := h.DB.Get(&forumSlug, `select forums.slug from forums
		inner join threads t on forums.slug = t.forum
	where t.id=$1 limit 1`, threadId); err != nil {
		ctx.SetStatusCode(500)
		return
	}
	currTime := time.Now().UTC()
	//con, _ := h.Conn.Acquire(context.Background())
	//defer con.Release()
	err = h.bulkInsert(postResult, forumSlug, threadId, currTime)
	if err != nil {
		if err, ok := err.(*pgconn.PgError); ok {
			switch err.Message {
			case "invalid parent id":
				ctx.SetStatusCode(409)
				data, _ := easyjson.Marshal(models.ConflictMsg)
				ctx.Write(data)
				return
			case "not found author":
				ctx.SetStatusCode(404)
				data, _ := easyjson.Marshal(models.NotFoundMsg)
				ctx.Write(data)
				return
			default:
				ctx.SetStatusCode(500)
				ctx.Write([]byte(err.Error()))
				return
			}
		}
	}

	retPosts := models.PostSet{}
	err = h.DB.
		Select(&retPosts, `select author, created, forum, id, isedited, message, thread, parent from posts where created=$1 and thread=$2 order by id`, currTime, threadId)
	if err == sql.ErrNoRows {
		data, _ := easyjson.Marshal(retPosts)
		ctx.Write(data)
		ctx.SetStatusCode(201)
		return
	}

	if err != nil {
		ctx.SetStatusCode(500)
		ctx.Write([]byte(err.Error()))
		return
	}

	ctx.SetStatusCode(201)

	data, _ := easyjson.Marshal(retPosts)
	ctx.Write(data)

}
func (h *Handler) GetPost(ctx *fasthttp.RequestCtx) {
	stringID := ctx.UserValue("id").(string)
	postID, err := strconv.Atoi(stringID)
	if err != nil {
		ctx.SetStatusCode(500)
		return
	}
	//var related []string
	paramsInStringByte := ctx.QueryArgs().Peek("related")

	related := strings.Split(string(paramsInStringByte), ",")

	retPost := models.Post{}
	//нету параметров
	if err := h.DB.Get(&retPost, `select author, created, forum, id, isedited, message, thread, parent from posts where id=$1`, postID); err != nil {
		switch {
		case err == sql.ErrNoRows:
			ctx.SetStatusCode(404)
			data, _ := easyjson.Marshal(models.NotFoundMsg)
			ctx.Write(data)
			return
		default:
			ctx.SetStatusCode(500)
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
				ctx.SetStatusCode(500)
				return
			}
		case "forum":
			if err := h.DB.Get(&forum, `select * from forums where lower(slug)=lower($1)`, retPost.Forum); err != nil {
				ctx.SetStatusCode(500)
				return
			}
		case "thread":
			if err := h.DB.Get(&thread, `select * from threads where id=$1`, retPost.Thread); err != nil {
				ctx.SetStatusCode(500)
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

	data, _ := json.Marshal(res)
	ctx.Write(data)

}
func (h *Handler) UpdatePost(ctx *fasthttp.RequestCtx) {
	stringID := ctx.UserValue("id").(string)
	postID, err := strconv.Atoi(stringID)
	if err != nil {
		ctx.SetStatusCode(500)
		return
	}

	retPost := models.Post{}
	newPostData := models.PostUpdate{}

	err = easyjson.Unmarshal(ctx.PostBody(), &newPostData)

	if err := h.DB.Get(&retPost, `select author, created, forum, id, isedited, message, thread, parent from posts where id=$1`, postID); err != nil {
		switch {
		case err == sql.ErrNoRows:
			ctx.SetStatusCode(404)
			data, _ := easyjson.Marshal(models.NotFoundMsg)
			ctx.Write(data)
			return
		default:
			ctx.SetStatusCode(500)
			return
		}
	}

	updatedPost := models.Post{}
	//то же самое сообщение
	if retPost.Message == newPostData.Message || newPostData.Message == "" {
		data, _ := easyjson.Marshal(retPost)
		ctx.Write(data)
		return
	}

	err = h.DB.Get(&updatedPost, `update posts set message=$1,
                 isedited=true where id=$2 returning author, created, 
                     forum, id, isedited, message, thread, parent`, newPostData.Message, postID)
	if err != nil {
		ctx.SetStatusCode(500)
		return
	}

	data, _ := easyjson.Marshal(updatedPost)
	ctx.Write(data)
}

func (h *Handler) GetAllPosts(ctx *fasthttp.RequestCtx) {
	//взяли thrID треда
	thrID, err := h.toID(ctx)
	if err != nil {
		ctx.SetStatusCode(404)
		data, _ := easyjson.Marshal(models.NotFoundMsg)
		ctx.Write(data)
		return
	}

	postList := models.PostSet{}

	params := &models.PostParams{}
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

	query := `select author, created, forum, id, isedited, message, thread, parent from posts where thread=$1 `

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
		resultPostList := models.PostSet{}
		var path []int
		switch {

		case params.Desc == false && params.Limit == 0 && params.Since == 0:
			_ = h.DB.Select(&resultPostList,
				`select author, created, forum, id, isedited, message, thread,parent from posts where thread=$1 order by path`, thrID)

		case params.Desc == true && params.Limit == 0 && params.Since == 0:
			_ = h.DB.Select(&resultPostList,
				`select author, created, forum, id, isedited, message, thread,parent from posts where thread=$1 order by path desc`, thrID)

		case params.Desc == false && params.Limit > 0 && params.Since == 0:
			_ = h.DB.Select(&resultPostList,
				`select author, created, forum, id, isedited, message, thread,parent from posts where thread=$1 order by path limit $2`, thrID, params.Limit)

		case params.Desc == false && params.Limit == 0 && params.Since > 0:
			err := h.Conn.QueryRow(context.Background(),
				`select path from posts where thread=$1 and path>(select path from posts where id=$2) order by path limit 1`,
				thrID, params.Since).Scan(&path)
			if err != nil {
				log.Println(err.Error())
			}
			_ = h.DB.Select(&resultPostList,
				`select author, created, forum, id, isedited, message, thread, parent 
						from posts 
						where thread=$1 and path>=$2
						order by path`,
				thrID, pq.Array(path))

		case params.Desc == false && params.Limit > 0 && params.Since > 0:
			err := h.Conn.QueryRow(context.Background(),
				`select path from posts where thread=$1 and path>(select path from posts where id=$2) order by path limit 1`,
				thrID, params.Since).Scan(&path)
			if err != nil {
				log.Println(err.Error())
			}
			_ = h.DB.Select(&resultPostList,
				`select author, created, forum, id, isedited, message, thread, parent 
						from posts 
						where thread=$1 and path>=$2
						order by path limit $3`,
				thrID, pq.Array(path), params.Limit)

		case params.Desc == true && params.Limit > 0 && params.Since == 0:
			_ = h.DB.Select(&resultPostList,
				`select author, created, forum, id, isedited, message, thread,parent from posts where thread=$1 order by path desc limit $2`, thrID, params.Limit)

		case params.Desc == true && params.Limit == 0 && params.Since > 0:
			err := h.Conn.QueryRow(context.Background(),
				`select  path from posts where thread=$1 and path<(select path from posts where id=$2)  order by path desc limit 1`,
				thrID, params.Since).Scan(&path)
			if err != nil {
				log.Println(err.Error())
			}
			_ = h.DB.Select(&resultPostList,
				`select author, created, forum, id, isedited, message, thread, parent 
						from posts 
						where thread=$1 and path<=$2
						order by path desc`,
				thrID, pq.Array(path), params.Limit)

		case params.Desc == true && params.Limit > 0 && params.Since > 0:
			err := h.Conn.QueryRow(context.Background(),
				`select  path from posts where thread=$1 and path<(select path from posts where id=$2)  order by path desc limit 1`,
				thrID, params.Since).Scan(&path)
			if err != nil {
				log.Println(err.Error())
			}
			_ = h.DB.Select(&resultPostList,
				`select author, created, forum, id, isedited, message, thread, parent 
						from posts 
						where thread=$1 and path<=$2
						order by path desc limit $3`,
				thrID, pq.Array(path), params.Limit)
		}

		data, _ := easyjson.Marshal(resultPostList)
		ctx.Write(data)
		return
	case "parent_tree":
		var path []int
		resultPostList := models.PostSet{}
		switch {
		case params.Desc == false && params.Limit == 0 && params.Since == 0:
			_ = h.DB.Select(&resultPostList,
				`select author, created, forum, id, isedited, message, thread,parent from posts where thread=$1 order by path`, thrID)

		case params.Desc == true && params.Limit == 0 && params.Since == 0:
			_ = h.DB.Select(&resultPostList,
				`select author, created, forum, id, isedited, message, thread,parent from posts where thread=$1 order by path [2] desc, path`, thrID)

		case params.Desc == false && params.Limit > 0 && params.Since == 0:
			//todo можно без in
			//нет смещения просто лимит пишем правильно
			query := `select author, created, forum, id, isedited, message, thread, parent
					from posts
					where thread = $1
  						and path[1:2] in (select path from posts where thread = $1 and cardinality(path) = 2 order by 1 limit $2)
					order by path`
			err = h.DB.Select(&resultPostList, query, thrID, params.Limit)

		case params.Desc == false && params.Limit == 0 && params.Since > 0:
			err := h.Conn.QueryRow(context.Background(),
				`select path from posts where thread=$1 and path>(select path from posts where id=$2) order by path limit 1`,
				thrID, params.Since).Scan(&path)
			if err != nil {
				log.Println(err.Error())
			}
			query := `select author, created, forum, id, isedited, message, thread, parent
					from posts where thread = $1 and path >= $2
						order by path`

			err = h.DB.Select(&resultPostList, query, thrID, pq.Array(path))

		case params.Desc == false && params.Limit > 0 && params.Since > 0:
			err = h.DB.Select(&resultPostList,
				`
					WITH our_path AS (
					    select path
					    from posts
					    where id = $2
					)
					select author, created, forum, id, isedited, message, thread, parent
					 from posts
					 where thread = $1
				   	 and path > (select path from our_path)
					 and path[1:2] in (select path from posts where thread = $1 and cardinality(path) = 2 and path[1:2] > (select path from our_path)[1:2] order by 1 limit $3)
					order by path`,
				thrID, params.Since, params.Limit)

		case params.Desc == true && params.Limit > 0 && params.Since == 0:
			h.DB.Select(&resultPostList, `select author, created, forum, id, isedited, message, thread, parent
					from posts
					where thread = $1
					  and path[1:2] in (select path
					                    from posts
					                    where thread = $1
					                      and cardinality(path) = 2
					                    order by 1 desc
					                    limit $2)
					order by path[2] desc, path`, thrID, params.Limit)

		case params.Desc == true && params.Limit == 0 && params.Since > 0:
			h.DB.Select(&resultPostList, `select author, created, forum, id, isedited, message, thread, parent
					from ((select *
						   from posts
						   where thread = $1
							 and path[1:2] = (select path from posts where id = $2)[1:2]
							 and path > (select path from posts where id = $2))
						  union
						  (select *
						   from posts
						   where thread = $1
							 and path[1:2] < (select path from posts where id = $2)[1:2])) sub
					order by path[2] desc, path`, thrID, params.Since)

		case params.Desc == true && params.Limit > 0 && params.Since > 0:
			h.DB.Select(&resultPostList, `select author, created, forum, id, isedited, message, thread, parent
					from ((select * --свой блок всегда выводим
						   from posts
						   where thread = $1
							 and path[1:2] = (select path from posts where id = $2)[1:2]
							 and path > (select path from posts where id = $2))
						  union
						  (select *
						   from posts
						   where thread = $1
							 and path[1:2] in (select path
											   from posts
											   where thread = $1
												 and cardinality(path) = 2
												 and path[1:2] < (select path from posts where id = $2)[1:2]
											   order by 1 desc
											   limit $3)))
								  sub
							order by path[2] desc, path;`, thrID, params.Since, params.Limit)
		}

		data, _ := easyjson.Marshal(resultPostList)
		ctx.Write(data)
		return
	default:
	}
	err = h.DB.Select(&postList, query, thrID)

	data, _ := easyjson.Marshal(postList)
	ctx.Write(data)
	return

}
func (h *Handler) toID(ctx *fasthttp.RequestCtx) (int, error) {
	slugOrID := ctx.UserValue("slug_or_id").(string)
	threadId := -1
	id, err := strconv.Atoi(slugOrID)
	if err != nil {
		slug := slugOrID
		if err := h.DB.Get(&threadId, "select id from threads where lower(slug)=lower($1)", slug); err != nil {
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
	if len(unsavedRows) == 0 {
		return nil
	}
	//inputRows := [][]interface{}{}
	batch := &pgx4.Batch{}
	for _, post := range unsavedRows {
		valueArgs := make([]interface{}, 6)
		valueArgs[0] = post.Author
		valueArgs[1] = created
		valueArgs[2] = slug
		valueArgs[3] = post.Message

		if post.Parent.Int64 != 0 {
			valueArgs[4] = post.Parent
		} else {
			valueArgs[4] = null.String{}
		}
		valueArgs[5] = threadID
		//inputRows = append(inputRows, valueArgs)
		batch.Queue(`insert into posts(author,created,forum,message,parent,thread) VALUES($1,$2,$3,$4,$5,$6)`, valueArgs...)
	}
	br := h.Conn.SendBatch(context.Background(), batch)
	defer br.Close()
	//_, err := h.Conn.CopyFrom(context.Background(), pgx4.Identifier{"posts"},
	//	[]string{"author", "created", "forum", "message", "parent", "thread"},
	//	pgx4.CopyFromRows(inputRows))

	if len(unsavedRows) != 100 {
		for i := 0; i < len(unsavedRows); i++ {
			_, err := br.Exec()
			if err != nil {
				return err
			}
		}
	}
	return nil
}
