// Package login implements a RESTful endpoint for performing a login.
// It accepts only a single POST method, and processes user credentials.
// Successful logins receive a time-limited session token which can be 
// used to perform authenticated operations. Repeated failed login attempts
// receive a retry penalty
package login

import (
  "context"
  "fmt"
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
  RouteName string = "login"
)


/*\
 *******************************************************************************
 *                              Type Definitions                               *
 *******************************************************************************
\*/


// Data: Login
type loginDataType struct {
  TimeFormat, UserTable, CredentialTable string
}

// Controller: Login
type Controller route.ControllerType[loginDataType]


/*\
 *******************************************************************************
 *                              Global Variables                               *
 *******************************************************************************
\*/


var loginData loginDataType = loginDataType {
  TimeFormat:      time.RFC3339,
  UserTable:       "actor",
  CredentialTable: "credential",
}


/*\
 *******************************************************************************
 *                                Constructors                                 *
 *******************************************************************************
\*/


func NewController (s route.Service) Controller {
  return Controller {
    Name:              RouteName,
    Methods: map[string]route.Method {
      http.MethodPost: route.Restful.Post,
    },
    Service:           s,
    Limit:             5 * time.Second,
    Data:              loginData,
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

type LoginCredential struct {
  Username        string `json:"username"`
  Passphrase      string `json:"passphrase"`
  Period          string `json:"period"`
}

type StoredCredential struct {
  Hash, Salt []byte
}

type SessionCredential struct {
  Secret     string `json:"secret"`
  Expiration string `json:"expiration"`
  Period     int64  `json:"period"` // Milliseconds
}

func (c *Controller) Post (x context.Context, rq *http.Request, re *route.Result) error {
  var (
    err     error               = nil
    ip      string              = x.Value(user.UserIPKey).(string)
    login   LoginCredential     = LoginCredential{}
  )

  // Expect JSON
  if login, err = route.ExpectJSON[LoginCredential](rq); nil != err {
    return re.ErrorWithStatus(err, http.StatusBadRequest)
  }

  // Check if a retry penalty exists (IP must exist)
  if c.Service.Auth.Penalised(ip) {
    return re.ErrorWithStatus(fmt.Errorf("Try again later"), http.StatusTooManyRequests)
  }

  // Extract stored login credentials
  q := fmt.Sprintf("SELECT b.hash, b.salt " +
                   "FROM %s AS a INNER JOIN %s AS b " +
		   "ON a.id = b.actor " +
		   "WHERE a.name = ?", 
		   c.Data.UserTable, c.Data.CredentialTable)

  // Define the authentication routine
  doAuth := func () (bool, error) {
    var stored StoredCredential

    rows, err := c.Service.Database.DB.Query(q, login.Username)
    if nil != err {
      return false, err
    }
    defer rows.Close()
    if !rows.Next() { // No error implies non-infrastructure related error
      return false, nil
    }
    if err = rows.Scan(&stored.Hash, &stored.Salt); nil != err {
      return false, err
    }
    return auth.Compare(login.Passphrase, stored.Salt, stored.Hash), nil
  }

  // Perform authentication 
  session, ok, err := c.Service.Auth.Authenticate(ip, login.Username, 
    login.Period, doAuth)
  if err != nil {
    // TODO: Don't leak info here
    return re.ErrorWithStatus(err, http.StatusInternalServerError)
  }

  // Wipe penalty and create session if OK; else penalise and return error
  if ok {
    c.Service.Auth.NoPenalty(ip)
  } else {
    c.Service.Auth.Penalise(ip)
    return re.ErrorWithStatus(fmt.Errorf("Bad credentials"), http.StatusUnauthorized)
  }

  // Compose response
  return re.Marshal(route.ContentTypeJSON,
    &SessionCredential {
      Secret:      session.Secret.HexString(),
      Expiration:  session.Expiration.Format(c.Data.TimeFormat),
      Period:      session.Period.Milliseconds(),
  })
}

func (c *Controller) Put (x context.Context, rq *http.Request, re *route.Result) error {
  return re.Unimplemented()
}

func (c *Controller) Delete (x context.Context, rq *http.Request, re *route.Result) error {
  return re.Unimplemented()
}

