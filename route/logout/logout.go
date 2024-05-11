package logout

import (
  "context"
  "encoding/json"
  "io/ioutil"
  "micrified.com/internal/user"
  "micrified.com/route"
  "micrified.com/service/auth"
  "net/http"
  "time"
)


// Data: Logout
type logoutData struct {}

// Controller: Logout
type Controller route.ControllerType[logoutData]


/*\
 *******************************************************************************
 *                            Interface: Controller                            *
 *******************************************************************************
\*/


func NewController (s route.Service) Controller {
  return Controller {
    Name:             "logout",
    Methods: map[string]route.Method {
      http.MethodPost: route.Restful.Post,
    },
    Service:           s,
    Limit:             5 * time.Second,
    Data:              logoutData{},
  }
}

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


/*\
 *******************************************************************************
 *                             Interface: Restful                              *
 *******************************************************************************
\*/


func (c *Controller) Get (x context.Context, rq *http.Request, re *route.Result) error {
  return re.Unimplemented()
}

type LogoutCredential struct {}

func (c *Controller) Post (x context.Context, rq *http.Request, re *route.Result) error {
  var (
    body   []byte                          = []byte{}
    err    error                           = nil
    ip     string                          = x.Value(user.UserIPKey).(string)
    logout auth.AuthData[LogoutCredential] = auth.AuthData[LogoutCredential]{}
  )

  fail := func(err error, status int) error {
    re.Status = status
    return err
  }

  // Read request body
  if body, err = ioutil.ReadAll(rq.Body); nil != err {
    return fail(err, http.StatusInternalServerError)
  }

  // Unmarshal to type
  if err = json.Unmarshal(body, &logout); nil != err {
    return fail(err, http.StatusBadRequest)
  }

  // Check if authorized
  if err = c.Service.Auth.Authorized(ip, logout.Username, logout.Secret); nil != err {
    return fail(err, http.StatusUnauthorized)
  }

  // Remove the session 
  if err = c.Service.Auth.Deauthenticate(logout.Username); nil != err {
    return fail(err, http.StatusInternalServerError)
  }

  // No content is to be returned, so HTTP status code 204 is expected
  return re.NoContent()
}

func (c *Controller) Put (x context.Context, rq *http.Request, re *route.Result) error {
  return re.Unimplemented()
}

func (c *Controller) Delete (x context.Context, rq *http.Request, re *route.Result) error {
  return re.Unimplemented()
}
