// Package logout provides a RESTful endpoint for performing a logout.
// It accepts only a single POST method, and requires a valid session
// token in order to process the logout request. A successful logout
// results in the session information for the authenticated agent being
// removed from the server session memory.
package logout

import (
  "context"
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
  RouteName string = "logout"
)


/*\
 *******************************************************************************
 *                              Type Definitions                               *
 *******************************************************************************
\*/


// Data: Logout
type logoutDataType struct {}

// Controller: Logout
type Controller route.ControllerType[logoutDataType]


/*\
 *******************************************************************************
 *                              Global Variables                               *
 *******************************************************************************
\*/


var logoutData logoutDataType = logoutDataType{}


/*\
 *******************************************************************************
 *                                Constructors                                 *
 *******************************************************************************
\*/


func NewController (s route.Service) Controller {
  return Controller {
    Name:             RouteName,
    Methods: map[string]route.Method {
      http.MethodPost: route.Restful.Post,
    },
    Service:           s,
    Limit:             5 * time.Second,
    Data:              logoutData,
  }
}


/*\
 *******************************************************************************
 *                            Interface: Controller                            *
 *******************************************************************************
\*/


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

type Post struct {}

func (c *Controller) Post (x context.Context, rq *http.Request, re *route.Result) error {
  var (
    err    error            = nil
    ip     string           = x.Value(user.UserIPKey).(string)
    post   auth.Frame[Post] = auth.Frame[Post]{}
  )

  // Expect JSON
  if post, err = route.ExpectJSON[auth.Frame[Post]](rq); nil != err {
    return re.ErrorWithStatus(err, http.StatusBadRequest)
  }

  // Check if authorized
  if err = c.Service.Auth.Authorized(ip, post.Username, post.Secret); nil != err {
    return re.ErrorWithStatus(err, http.StatusUnauthorized)
  }

  // Remove the session 
  if err = c.Service.Auth.Deauthenticate(post.Username); nil != err {
    return re.ErrorWithStatus(err, http.StatusInternalServerError)
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
