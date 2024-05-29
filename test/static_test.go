
package main

import (
  "bytes"
  "encoding/json"
  "fmt"
  "io/ioutil"
  "log"
  "micrified.com/route"
  "micrified.com/route/static"
  "micrified.com/route/login"
  "micrified.com/route/logout"
  "micrified.com/service/auth"
  "net/http"
  "os"
  "testing"
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
  LoginURL  string = os.Getenv("TEST_HOSTNAME") + "/" + login.RouteName
  LogoutURL string = os.Getenv("TEST_HOSTNAME") + "/" + logout.RouteName
  StaticURL string = os.Getenv("TEST_HOSTNAME") + "/" + static.RouteName
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

// TestStatic performs a login, then tests the following REST operations
// of static: LOGIN + POST + GET + PUT + GET + DELETE + LOGOUT
func TestStatic (t *testing.T) {

  // Environment variables
  username   := os.Getenv("TEST_USERNAME")
  passphrase := os.Getenv("TEST_PASSPHRASE")

  // Static page to test with
  name, path := "test", StaticURL + "/test"

  // getStatic performs a GET request to the static endpoint
  getStatic := func(name string, getResponse *static.GetResponse) error {
    req, err := http.NewRequest(http.MethodGet, StaticURL + "/" + name, nil)
    if nil != err {
      return err
    }
    res, err := http.DefaultClient.Do(req)
    if nil != err {
      return err
    }
    if http.StatusOK != res.StatusCode {
      return fmt.Errorf("Bad status (got %d, expected %d)", res.StatusCode,
        http.StatusOK)
    }
    body, err := ioutil.ReadAll(res.Body)
    if nil != err {
      return err
    }
    return json.Unmarshal(body, getResponse)
  }

  // LOGIN
  loginFunc := Request[login.SessionCredential, login.LoginCredential]
  loginPost, sessionCredential := login.LoginCredential {
    Username:   username,
    Passphrase: passphrase,
  }, login.SessionCredential{}
  err := loginFunc(LoginURL, http.MethodPost, http.StatusOK, loginPost, &sessionCredential)
  if nil != err {
    t.Fatalf("Login failed: %v", err)
  }

  // POST
  postFunc := Request[any, auth.AuthData[static.Post]]
  post := auth.AuthData[static.Post] {
    Username: username,
    Secret:   sessionCredential.Secret,
    Data: static.Post {
      Body:    "temporary static page",
    },
  }
  err = postFunc(path, http.MethodPost, http.StatusOK, post, nil)
  if nil != err {
    t.Fatalf("POST failed: %v", err)
  }

  // GET
  getResponse := static.GetResponse{}
  err = getStatic(name, &getResponse)
  if nil != err {
    t.Fatalf("GET failed: %v", err)
  }
  if getResponse.Body != post.Data.Body {
    t.Fatalf("GET response not as expected")
  }

  // PUT
  putFunc := Request[any, auth.AuthData[static.Put]]
  put := auth.AuthData[static.Put] {
    Username:   username,
    Secret: sessionCredential.Secret,
    Data: static.Put {
      Body: "temporary static page update",
    },
  }
  err = putFunc(path, http.MethodPut, http.StatusOK, put, nil)

  if nil != err {
    t.Fatalf("PUT failed: %v", err)
  }

  // GET
  err = getStatic(name, &getResponse)
  if nil != err {
    t.Fatalf("GET failed: %v", err)
  }
  if put.Data.Body != getResponse.Body {
    t.Fatalf("GET blog content not as expected!")
  }

  // DELETE
  delFunc := Request[any, auth.AuthData[static.Delete]]
  del := auth.AuthData[static.Delete] {
    Username:   username,
    Secret: sessionCredential.Secret,
    Data: static.Delete{},
  }
  err = delFunc(path, http.MethodDelete, http.StatusOK, del, nil)
  if nil != err {
    t.Fatalf("DELETE Response not as expected: %v", err)
  }

  // LOGOUT
  logoutFunc := Request[any, auth.AuthData[logout.LogoutCredential]]
  logoutPost := auth.AuthData[logout.LogoutCredential]{
    Username: username,
    Secret:   sessionCredential.Secret,
    Data:     logout.LogoutCredential{},
  }
  err = logoutFunc(LogoutURL, http.MethodPost, http.StatusNoContent, logoutPost, nil)
  if nil != err {
    t.Fatalf("LOGOUT Response not as expected: %v", err)
  }

}

