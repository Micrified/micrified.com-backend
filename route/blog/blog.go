// Package blog implements a RESTful endpoint for blogs.
// It supports the creation, modification, and deletion of blog posts
package blog

import (
  "context"
  "database/sql"
  "fmt"
  "micrified.com/internal/user"
  "micrified.com/route"
  "micrified.com/service/auth"
  "net/http"
  "strconv"
  "time"
)


/*\
 *******************************************************************************
 *                                  Constants                                  *
 *******************************************************************************
\*/


const (
  RouteName string     = "blog"
  RouteListName string = "blogs"
)


/*\
 *******************************************************************************
 *                              Type Definitions                               *
 *******************************************************************************
\*/


// Data: Blog
type blogDataType struct {
  TimeFormat, IndexTable, ContentTable string
}

// Controller: Blog
type Controller route.ControllerType[blogDataType]

// ListController: Blog
type ListController route.ControllerType[blogDataType]


/*\
 *******************************************************************************
 *                              Global Variables                               *
 *******************************************************************************
\*/


var blogData blogDataType = blogDataType {
  TimeFormat:   "2006-01-02 15:04:05",
  IndexTable:   "blog",
  ContentTable: "page",
}


/*\
 *******************************************************************************
 *                                Constructors                                 *
 *******************************************************************************
\*/


func NewListController (s route.Service) ListController {
  return ListController {
    Name: RouteListName,
    Methods: map[string]route.Method {
      http.MethodGet: route.Restful.Get,
    },
    Service: s,
    Limit:   5 * time.Second,
    Data:    blogData,
  }
}

func NewController (s route.Service) Controller {
  return Controller {
    Name:                RouteName,
    Methods: map[string]route.Method {
      http.MethodGet:    route.Restful.Get,
      http.MethodPost:   route.Restful.Post,
      http.MethodPut:    route.Restful.Put,
      http.MethodDelete: route.Restful.Delete,
    },
    Service:             s,
    Limit:               5 * time.Second,
    Data:                blogData,
  }
}


/*\
 *******************************************************************************
 *                            Interface: Controller                            *
 *******************************************************************************
\*/


// Controller

func (c *Controller) Route () string {
  return "/" + c.Name
}

func (c *Controller) Handler (s string) route.Method {
  if method, ok := c.Methods[s]; ok {
    return method
  }
  return nil
}

func (c *Controller) Timeout () time.Duration {
  return c.Limit
}


// ListController

func (c *ListController) Route () string {
  return "/" + c.Name
}

func (c *ListController) Handler (s string) route.Method {
  if method, ok := c.Methods[s]; ok {
    return method
  }
  return nil
}

func (c *ListController) Timeout () time.Duration {
  return c.Limit
}


/*\
 *******************************************************************************
 *                             Interface: Restful                              *
 *******************************************************************************
\*/


// Controller

type BlogResponse struct {
  ID       string `json:"id"`
  Title    string `json:"title"`
  Subtitle string `json:"subtitle"`
  Body     string `json:"body"`
  Created  string `json:"created"`
  Updated  string `json:"updated"`
}

func (c *Controller) Get (x context.Context, rq *http.Request, re *route.Result) error {
  var (
    blog    BlogResponse = BlogResponse{}
    blog_id int          = -1
    err     error        = nil
  )

  q := fmt.Sprintf("SELECT a.id, a.title, a.subtitle, b.body, b.created, b.updated " + 
                   "FROM %s AS a INNER JOIN %s AS b " +
		   "ON a.page = b.id " + 
		   "WHERE a.id = ?", c.Data.IndexTable, c.Data.ContentTable)

  // Validate ID
  if blog_id, err = strconv.Atoi(rq.URL.Query().Get("id")); nil != err {
    return re.ErrorWithStatus(
      fmt.Errorf("Invalid query parameter"), http.StatusBadRequest)
  }

  // Extract row
  rows, err := c.Service.Database.DB.Query(q, blog_id)
  if nil != err {
    return re.ErrorWithStatus(err, http.StatusInternalServerError)
  }
  defer rows.Close()

  // Verify entry exists
  if !rows.Next() {
    return re.ErrorWithStatus(
      fmt.Errorf("Blog %s not found", blog_id), http.StatusNotFound)
  }

  // Marshal rows
  if err = rows.Scan(&blog.ID, &blog.Title, &blog.Subtitle, &blog.Body, 
    &blog.Created, &blog.Updated); nil != err {
    return re.ErrorWithStatus(err, http.StatusInternalServerError)
  }

  // Write to buffer and return any encoding error
  return re.Marshal(route.ContentTypeJSON, &blog)
}

type Post struct {
  Title    string `json:"title"`
  Subtitle string `json:"subtitle"`
  Body     string `json:"body"`
}

type PostResponse struct {
  ID       string `json:"id"`
  Title    string `json:"title"`
  Subtitle string `json:"subtitle"`
  Body     string `json:"body"`
  Created  string `json:"created"`
  Updated  string `json:"updated"`
}

func (c *Controller) Post (x context.Context, rq *http.Request, re *route.Result) error {
  var (
    err       error               = nil
    ip        string              = x.Value(user.UserIPKey).(string)
    post      auth.Frame[Post] = auth.Frame[Post]{}
    timeStamp time.Time           = time.Now().UTC().Truncate(time.Second)
  )

  // Expect JSON
  if post, err = route.ExpectJSON[auth.Frame[Post]](rq); nil != err {
    return re.ErrorWithStatus(err, http.StatusBadRequest)
  }

  // Check if authorized
  if err = c.Service.Auth.Authorized(ip, post.Username, post.Secret); nil != err {
    return re.ErrorWithStatus(err, http.StatusUnauthorized)
  }
    
  // Define insert content
  insertBody := func (lastResult sql.Result, t *sql.Tx) (sql.Result, error) {
    q := fmt.Sprintf("INSERT INTO %s (created,updated,body) VALUES (?,?,?)",
      c.Data.ContentTable)
    return t.ExecContext(c.Service.Database.Context, q, timeStamp, timeStamp,
      post.Data.Body)
  }

  // Define insert record
  insertRecord := func (lastResult sql.Result, t *sql.Tx) (sql.Result, error) {
    id, err := lastResult.LastInsertId()
    if nil != err {
      return nil, err
    }
    q := fmt.Sprintf("INSERT INTO %s (title,subtitle,page) VALUES (?,?,?)", 
      c.Data.IndexTable)
    return t.ExecContext(c.Service.Database.Context, q, post.Data.Title, 
      post.Data.Subtitle, id)
  }

  // Execute sequenced insert operations; get back result
  r, err := c.Service.Database.Transaction(insertBody, insertRecord)
  if nil != err {
    return re.ErrorWithStatus(err, http.StatusInternalServerError)
  }

  // Verify the right number of rows were affected
  n, err := r.RowsAffected()
  if nil != err {
    return re.ErrorWithStatus(err, http.StatusInternalServerError)
  } else if 1 != n {
    return re.ErrorWithStatus(
      fmt.Errorf("Unexpected result (expected 1 row affected, got %d)", n),
      http.StatusInternalServerError)
  }

  // Get the record ID
  id, err := r.LastInsertId()
  if nil != err {
    return re.ErrorWithStatus(err, http.StatusInternalServerError)
  }

  // Write to buffer and return any encoding error
  return re.Marshal(route.ContentTypeJSON, 
    &PostResponse {
      ID:       strconv.FormatInt(id, 10),
      Title:    post.Data.Title,
      Subtitle: post.Data.Subtitle,
      Body:     post.Data.Body,
      Created:  timeStamp.Format(c.Data.TimeFormat),
      Updated:  timeStamp.Format(c.Data.TimeFormat),
    })
}

type Put struct {
  ID       string `json:"id"`
  Title    string `json:"title"`
  Subtitle string `json:"subtitle"`
  Body     string `json:"body"`
}

type PutResponse struct {
  ID       string `json:"id"`
  Title    string `json:"title"`
  Subtitle string `json:"subtitle"`
  Updated  string `json:"updated"`
  Body     string `json:"body"`
}

func (c *Controller) Put (x context.Context, rq *http.Request, re *route.Result) error {
  var (
    err       error              = nil
    ip        string             = x.Value(user.UserIPKey).(string)
    put       auth.Frame[Put] = auth.Frame[Put]{}
    timeStamp time.Time          = time.Now().UTC().Truncate(time.Second)
  )

  // Define update record
  updateRecord := func (lastResult sql.Result, conn *sql.Conn) (sql.Result, error) {
    q := fmt.Sprintf("UPDATE %s AS a INNER JOIN %s AS b ON a.page = b.id " +
                     "SET a.title = ?, a.subtitle = ?, b.updated = ?, b.body = ? " +
		     "WHERE a.id = ?", c.Data.IndexTable, c.Data.ContentTable)
    return conn.ExecContext(c.Service.Database.Context, q, put.Data.Title, 
      put.Data.Subtitle, timeStamp, put.Data.Body, put.Data.ID)
  }

  // Expect JSON
  if put, err = route.ExpectJSON[auth.Frame[Put]](rq); nil != err {
    return re.ErrorWithStatus(err, http.StatusBadRequest)
  }

  // Check if authorized
  if err = c.Service.Auth.Authorized(ip, put.Username, put.Secret); nil != err {
    return re.ErrorWithStatus(err, http.StatusUnauthorized)
  }

  // Execute sequenced connection operations; get back result
  _, err = c.Service.Database.Connection(updateRecord)
  if nil != err {
    return re.ErrorWithStatus(err, http.StatusInternalServerError)
  }

  // Don't verify rows affected (could be none if no change made)

  // No difference is needed here in the return type
  return re.Marshal(route.ContentTypeJSON,
    &PutResponse {
      ID:       put.Data.ID,
      Title:    put.Data.Title,
      Subtitle: put.Data.Subtitle,
      Updated:  timeStamp.Format(c.Data.TimeFormat),
      Body:     put.Data.Body,
  })
}

type Delete struct {
  ID string `json:"id"`
}

func (c *Controller) Delete (x context.Context, rq *http.Request, re *route.Result) error {
  var (
    err  error                 = nil
    ip   string                = x.Value(user.UserIPKey).(string)
    del  auth.Frame[Delete] = auth.Frame[Delete]{}
  )

  // Define delete record
  deleteRecord := func (lastResult sql.Result, conn *sql.Conn) (sql.Result, error) {
    q := fmt.Sprintf("DELETE a, b FROM %s AS a INNER JOIN %s AS b " +
                     "ON a.page = b.id " +
                     "WHERE a.id = ?", c.Data.IndexTable, c.Data.ContentTable)
    return conn.ExecContext(c.Service.Database.Context, q, del.Data.ID)
  }

  // Expect JSON
  if del, err = route.ExpectJSON[auth.Frame[Delete]](rq); nil != err {
    return re.ErrorWithStatus(err, http.StatusBadRequest)
  }

  // Check if authorized
  if err = c.Service.Auth.Authorized(ip, del.Username, del.Secret); nil != err {
    return re.ErrorWithStatus(err, http.StatusUnauthorized)
  }

  // Execute sequenced connection operations; get back result
  r, err := c.Service.Database.Connection(deleteRecord)
  if nil != err {
    return re.ErrorWithStatus(err, http.StatusInternalServerError)
  }

  // Verify the right number of rows were affected
  n, err := r.RowsAffected()
  if nil != err {
    return re.ErrorWithStatus(err, http.StatusInternalServerError)
  } else if 2 != n {
    return re.ErrorWithStatus(
      fmt.Errorf("Unexpected result (expected 2 rows affected, got %d)", n), 
      http.StatusInternalServerError)
  }

  return nil
}


// ListController

type BlogHeader struct {
  ID       string `json:"id"`
  Title    string `json:"title"`
  Subtitle string `json:"subtitle"`
  Created  string `json:"created"`
  Updated  string `json:"updated"`
}

func (c *ListController) Get (x context.Context, rq *http.Request, re *route.Result) error {
  var (
    head BlogHeader
    list []BlogHeader
  )
  q := fmt.Sprintf("SELECT a.id, a.title, a.subtitle, b.created, b.updated " +
                   "FROM %s AS a INNER JOIN %s AS b " + 
                   "ON a.page = b.id " +
                   "ORDER BY b.created", c.Data.IndexTable, c.Data.ContentTable)

  // Extract rows
  rows, err := c.Service.Database.DB.Query(q)
  if nil != err {
    return re.ErrorWithStatus(err, http.StatusInternalServerError)
  }
  defer rows.Close()

  // Marshal rows
  for rows.Next() {
    if err = rows.Scan(&head.ID, &head.Title, &head.Subtitle, &head.Created,
      &head.Updated); nil != err {
        break
      } else {
        list = append(list, head)
      }
  }

  // Check error
  if nil != err {
    return re.ErrorWithStatus(err, http.StatusInternalServerError)
  }

  // Write to buffer and return any encoding error
  return re.Marshal(route.ContentTypeJSON, &list)
}

func (c *ListController) Post (x context.Context, rq *http.Request, re *route.Result) error {
  return re.Unimplemented()
}


func (c *ListController) Put (x context.Context, rq *http.Request, re *route.Result) error {
  return re.Unimplemented()
}

func (c *ListController) Delete (x context.Context, rq *http.Request, re *route.Result) error {
  return re.Unimplemented()
}
