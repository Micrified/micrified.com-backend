package main

import (
  "bytes"
  "encoding/json"
  "fmt"
  "io/ioutil"
  "log"
  "micrified.com/route"
  "micrified.com/route/login"
  "micrified.com/route/logout"
  "micrified.com/service/auth"
  "net/http"
  "os"
  "strconv"
  "testing"
  "time"
)


/*\
 *******************************************************************************
 *                              Generic Functions                              *
 *******************************************************************************
\*/


// Request is a generic function template 
func Request [T any, U any] (url, method string, status int, u U, t *T) error {
  var (
    body   []byte         = []byte{}
    buffer bytes.Buffer   = bytes.Buffer{}
    req    *http.Request  = nil
    res    *http.Response = nil
    err    error          = nil
  )

  // Marshal and send request
  if err = json.NewEncoder(&buffer).Encode(u); nil != err {
    return err
  }
  if req, err = http.NewRequest(method, url, bytes.NewBuffer(buffer.Bytes())); nil != err {
    return err
  }
  req.Header.Set(route.ContentTypeName, route.ContentTypeJSON)

  // Get response and unmarshal if required
  if res, err = http.DefaultClient.Do(req); nil != err {
    return err
  }
  defer res.Body.Close()
  if status != res.StatusCode {
    return fmt.Errorf("Bad status (got %d, expected %d)", res.StatusCode, status)
  }
  if nil != t {
    if body, err = ioutil.ReadAll(res.Body); nil != err {
      return err
    }
    if err = json.Unmarshal(body, t); nil != err {
      return err
    }
  }
  return nil
}


/*\
 *******************************************************************************
 *                                    Tests                                    *
 *******************************************************************************
\*/


var (
  LoginURL string = os.Getenv("TEST_HOSTNAME") + "/" + login.RouteName
  LogoutURL string = os.Getenv("TEST_HOSTNAME") + "/" + logout.RouteName
)


func TestMain (m *testing.M) {
  vs := []string{"TEST_HOSTNAME", "TEST_USERNAME", "TEST_PASSPHRASE"}
  for _, v := range vs {
    if "" == os.Getenv(v) {
      log.Fatalf("Unset environment variable: %s", v)
    }
  }
  os.Exit(m.Run())
}


// TestLoginPenalty verifies that repeated login requests with incorrect
// credentials accrues an IP based penalty. It verifies the penalty is 
// wiped when a correct login occurs after the penalty expires
//
// Note: Ensure server is initialized with same auth configuration!
func TestLoginPenalty (t *testing.T) {
  loginFunc := Request[login.SessionCredential, login.LoginCredential]
  penaltyConfig := auth.Config {
    Base: 2, Factor: 2, Limit: 8, Retry: 3,
  }

  // Environment variables
  validCredentials, invalidCredentials := login.LoginCredential {
    Username: os.Getenv("TEST_USERNAME"), Passphrase: os.Getenv("TEST_PASSPHRASE"),
  }, login.LoginCredential {
    Username: "", Passphrase: "",
  }

  // [Retry Limit] Verify that a client attempting to login has 
  // a number of penalty free retries. 
  testRetries := func(out *login.SessionCredential) (err error) {
    // [Incorrect] Attempt one incorrect login and all retries
    for i := 0; i < (1 + penaltyConfig.Retry) && nil == err; i++ {
      err = loginFunc(LoginURL, http.MethodPost, http.StatusUnauthorized, 
        invalidCredentials, nil)
    }
    if nil != err {
      return
    }
    // [Correct] Login should be permitted without penalty
    err = loginFunc(LoginURL, http.MethodPost, http.StatusOK,
      validCredentials, out)
    return
  }

  // Test: Retry mechanism allows limited penalty-free login
  fmt.Println(LoginURL)
  sessionCredential1 := login.SessionCredential{}
  if err := testRetries(&sessionCredential1); nil != err {
    t.Fatalf("Retries failed: %v", err)
  }

  // Test: Correct login has reset retry mechanism
  sessionCredential2 := login.SessionCredential{}
  if err := testRetries(&sessionCredential2); nil != err {
    t.Fatalf("Retries reset failed: %v", err)
  }
}

// TestSessionMutex tests the mututal exclusivity of sessions. It checks that
// a new login returns a new unique session token, and that the old session
// token is no longer valid
func TestSessionMutex (t *testing.T) {
  loginFunc, logoutFunc := Request[login.SessionCredential, login.LoginCredential],
                           Request[any,  auth.AuthData[logout.LogoutCredential]]
  validCredentials := login.LoginCredential {
    Username: os.Getenv("TEST_USERNAME"), Passphrase: os.Getenv("TEST_PASSPHRASE"),
  }

  // Number of sessions to test
  n := 4

  sessions := make([]login.SessionCredential, n)
  for i := 0; i < n; i++ {
    // Login
    err := loginFunc(LoginURL, http.MethodPost, http.StatusOK,
      validCredentials, &sessions[i])
    if nil != err {
      t.Fatalf("Login request error: %v", err)
    }
    // Try to logout using previous session credentials (shouldn't be possible)
    if i > 1 {
      logoutPost := auth.AuthData[logout.LogoutCredential] {
        Username: validCredentials.Username,
        Secret:   sessions[i-1].Secret,
        Data:     logout.LogoutCredential{},
      }
      err = logoutFunc(LogoutURL, http.MethodPost, http.StatusUnauthorized,
        logoutPost, nil)
      if nil != err {
        t.Fatalf("Logout request error: %v", err)
      }
    }
  }

  // Logout should be possible with the last session credential
  logoutPost := auth.AuthData[logout.LogoutCredential] {
    Username: validCredentials.Username,
    Secret:   sessions[n-1].Secret,
    Data:     logout.LogoutCredential{},
  }
  err := logoutFunc(LogoutURL, http.MethodPost, http.StatusNoContent, 
    logoutPost, nil)
  if nil != err {
    t.Fatalf("Logout request error: %v", err)
  }
}

// TestSessionExpiration tests that session tokens are not reusable after the
// expiration deadline. 
func TestSessionExpiration (t *testing.T) {
  loginFunc, logoutFunc := Request[login.SessionCredential, login.LoginCredential],
                           Request[any, auth.AuthData[logout.LogoutCredential]]
  period := 5
  validCredentials := login.LoginCredential {
    Username:   os.Getenv("TEST_USERNAME"),
    Passphrase: os.Getenv("TEST_PASSPHRASE"),
    Period:     strconv.Itoa(period),
  }

  // Login
  sessionCredential := login.SessionCredential{}
  err := loginFunc(LoginURL, http.MethodPost, http.StatusOK, validCredentials,
   &sessionCredential)
  if nil != err {
    t.Fatalf("Login request error: %v", err)
  }

  // Wait until the period expires
  time.Sleep(time.Duration(period) * time.Second)

  // Logout
  logoutPost := auth.AuthData[logout.LogoutCredential] {
    Username: validCredentials.Username,
    Secret:   sessionCredential.Secret,
    Data:     logout.LogoutCredential{},
  }
  err = logoutFunc(LogoutURL, http.MethodPost, http.StatusUnauthorized,
    logoutPost, nil)
  if nil != err {
    t.Fatalf("Session valid past expiration: %v", err)
  }
}

