// Package static implements a RESTful endpoint for named pages.
// It handles all methods along path: /static/{name} where name is a single
// segment with no forward-slash path separator.
package static

import (
  "context"
  "database/sql"
  "encoding/json"
  "fmt"
  "io/ioutil"
  "micrified.com/internal/user"
  "micrified.com/route"
  "micrified.com/service/auth"
  "net/http"
  "time"
)


/*\
 *******************************************************************************
 *                                  Constants                                  *
 *******************************************************************************
\*/


const (
  RouteName string = "static"
)


/*\
 *******************************************************************************
 *                              Type Definitions                               *
 *******************************************************************************
\*/


// Data: Static
type staticDataType struct {
  TimeFormat, IndexTable, ContentTable string
}

// Controller: Static
type Controller route.ControllerType[staticDataType]


/*\
 *******************************************************************************
 *                              Global Variables                               *
 *******************************************************************************
\*/


var staticData staticDataType = staticDataType {
  TimeFormat:   "2006-01-02 15:04:05",
  IndexTable:   "static_pages",
  ContentTable: "page_content",
}


/*\
 *******************************************************************************
 *                                Constructors                                 *
 *******************************************************************************
\*/


func NewController (s route.Service) Controller {
  return Controller {
    Name:    RouteName,
    Methods: map[string]route.Method {
      http.MethodGet: route.Restful.Get,
    },
    Service: s,
    Limit:   5 * time.Second,
    Data:    staticData,
  }
}


/*\
 *******************************************************************************
 *                            Interface: Controller                            *
 *******************************************************************************
\*/


func (c *Controller) Route () string {
  return "/" + c.Name + "/{name}"
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


/*\
 *******************************************************************************
 *                             Interface: Restful                              *
 *******************************************************************************
\*/


type GetResponse struct {
  Body    string `json:"body"`
  Created string `json:"created"`
  Updated string `json:"updated"`
}

func (c *Controller) Get (x context.Context, rq *http.Request, re *route.Result) error {
  var (
    page GetResponse = GetResponse{}
    name string      = rq.PathValue("name")
    err  error       = nil
  )

  q := fmt.Sprintf("SELECT a.body, a.created, a.updated FROM %s AS a " +
                   "INNER JOIN %s AS b " +
		   "ON a.id = b.content_id " +
		   "WHERE b.url_hash = unhex(md5(?))",
		   c.Data.ContentTable, c.Data.IndexTable)

  // Extract row
  rows, err := c.Service.Database.DB.Query(q, name)
  if nil != err {
    return re.ErrorWithStatus(err, http.StatusInternalServerError)
  }
  defer rows.Close()

  // Verify entry exists
  if !rows.Next() {
    return re.ErrorWithStatus(fmt.Errorf("Page %s not found", name), http.StatusNotFound)
  }

  // Marshal rows
  if err = rows.Scan(&page.Body, &page.Created, &page.Updated); nil != err {
    return re.ErrorWithStatus(err, http.StatusInternalServerError)
  }

  return re.Marshal(route.ContentTypeJSON, &page)
}

type StaticPost struct {
  Name string `json:"name"`
  Body string `json:"body"`
}

type StaticPostResponse struct {
  Name    string `json:"name"`
  Body    string `json:"body"`
  Created string `json:"created"`
  Updated string `json:"updated"`
}

func (c *Controller) Post (x context.Context, rq *http.Request, re *route.Result) error {
  var (
    body      []byte                    = []byte{}
    err       error                     = nil
    ip        string                    = x.Value(user.UserIPKey).(string)
    post      auth.AuthData[StaticPost] = auth.AuthData[StaticPost]{}
    timeStamp time.Time                 = time.Now().UTC().Truncate(time.Second)
  )

  // Read request body
  if body, err = ioutil.ReadAll(rq.Body); nil != err {
    return re.ErrorWithStatus(err, http.StatusInternalServerError)
  }

  // Unmarshal to type
  if err = json.Unmarshal(body, &post); nil != err {
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
    q := fmt.Sprintf("INSERT INTO %s (url_hash,content_id) " +
                     "VALUES (UNHEX(MD5(?)),?)",
		     c.Data.IndexTable)
    return t.ExecContext(c.Service.Database.Context, q, post.Data.Name, id)
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

  // Write to buffer and return any encoding error
  return re.Marshal(route.ContentTypeJSON,
    &StaticPostResponse {
      Name:    post.Data.Name,
      Body:    post.Data.Body,
      Created: timeStamp.Format(c.Data.TimeFormat),
      Updated: timeStamp.Format(c.Data.TimeFormat),
  })
}

type StaticPut struct {
  Name string `json:"name"`
  Body string `json:"body"`
}

type StaticPutResponse struct {
  Name    string `json:"name"`
  Body    string `json:"body"`
  Updated string `json:"updated"`
}

func (c *Controller) Put (x context.Context, rq *http.Request, re *route.Result) error {
  var (
    body      []byte                   = []byte{}
    err       error                    = nil
    ip        string                   = x.Value(user.UserIPKey).(string)
    put       auth.AuthData[StaticPut] = auth.AuthData[StaticPut]{}
    timeStamp time.Time                = time.Now().UTC().Truncate(time.Second)
  )

  // Define update record
  updateRecord := func (lastResult sql.Result, conn *sql.Conn) (sql.Result, error) {
    q := fmt.Sprintf("UPDATE %s AS a INNER JOIN %s AS b " +
                     "ON a.content_id = b.id " +
		     "SET b.body = ? " +
		     "WHERE a.url_hash = UNHEX(MD5(?))", 
		     c.Data.IndexTable, c.Data.ContentTable)
    return conn.ExecContext(c.Service.Database.Context, q, put.Data.Body,
      put.Data.Name)
  }

  // Read request body
  if body, err = ioutil.ReadAll(rq.Body); nil != err {
    return re.ErrorWithStatus(err, http.StatusInternalServerError)
  }

  // Unmarshal to type
  if err = json.Unmarshal(body, &put); nil != err {
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
    &StaticPutResponse {
      Name:    put.Data.Name,
      Body:    put.Data.Body,
      Updated: timeStamp.Format(c.Data.TimeFormat),
  })
}

type StaticDelete struct {
  Name string `json:"name"`
}

func (c *Controller) Delete (x context.Context, rq *http.Request, re *route.Result) error {
  var (
    body []byte                      = []byte{}
    err  error                       = nil
    ip   string                      = x.Value(user.UserIPKey).(string)
    del  auth.AuthData[StaticDelete] = auth.AuthData[StaticDelete]{}
  )

  // Define delete record
  deleteRecord := func (lastResult sql.Result, conn *sql.Conn) (sql.Result, error) {
    q := fmt.Sprintf("DELETE a, b FROM %s AS a INNER JOIN %s AS b " +
                     "ON a.content_id = b.id " +
		     "WHERE a.url_hash = UNHEX(MD5(?))",
		     c.Data.IndexTable, c.Data.ContentTable)
    return conn.ExecContext(c.Service.Database.Context, q, del.Data.Name)
  }

  // Read request body
  if body, err = ioutil.ReadAll(rq.Body); nil != err {
    return re.ErrorWithStatus(err, http.StatusInternalServerError)
  }

  // Unmarshal to type
  if err = json.Unmarshal(body, &del); nil != err {
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

