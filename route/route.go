package route

import (
  "bytes"
  "context"
  "encoding/json"
  "fmt"
  "io/ioutil"
  "micrified.com/service/auth"
  "micrified.com/service/database"
  "net/http"
  "time"
)

const (
  ContentTypeName  = "Content-Type"
  ContentTypeJSON  = "application/json"
  ContentTypePlain = "text/plain"
)


/*\
 *******************************************************************************
 *                        Definition: Helper Functions                         *
 *******************************************************************************
\*/


func ExpectJSON [T any] (rq *http.Request) (T, error) {
  var (
    body []byte
    err  error
    data T
  )
  if body, err = ioutil.ReadAll(rq.Body); nil != err {
    return data, err
  }
  if err = json.Unmarshal(body, &data); nil != err {
    return data, err
  }
  return data, nil
}


/*\
 *******************************************************************************
 *                             Definition: Result                              *
 *******************************************************************************
\*/


type Result struct {
  Buffer      bytes.Buffer
  ContentType string
  Status      int
}

func (re *Result) Marshal (contentType string, p any) error {
  re.ContentType = contentType
  return json.NewEncoder(&re.Buffer).Encode(p)
}

func (re *Result) ErrorWithStatus (err error, status int) error {
  re.Status = status
  return err
}

func (re *Result) Unimplemented () error {
  re.Status = http.StatusNotImplemented
  return fmt.Errorf("Invalid API call")
}

func (re *Result) NoContent () error {
  re.Status = http.StatusNoContent
  return nil
}

func DefaultResult () Result {
  return Result {
    Buffer:      bytes.Buffer{},
    ContentType: ContentTypePlain,
    Status:      http.StatusOK,
  }
}


/*\
 *******************************************************************************
 *                       Definition: Controller, Restful                       *
 *******************************************************************************
\*/


// Restful interface
type Restful interface {
  Get(context.Context, *http.Request, *Result) error
  Post(context.Context, *http.Request, *Result) error
  Put(context.Context, *http.Request, *Result) error
  Delete(context.Context, *http.Request, *Result) error
}

// HTTP method type
type Method func (Restful, context.Context, *http.Request, *Result) error

// Controller interface
type Controller interface {
  Route() string
  Handler(string) Method
  Timeout() time.Duration
  Restful
}

// Service structure
type Service struct {
  Auth *auth.Service
  Database *database.Service
}

// Templated controller type generator
type ControllerType [T any] struct {
  Name    string
  Methods map[string]Method
  Service Service
  Limit   time.Duration
  Data T
}

