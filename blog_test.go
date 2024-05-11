package main

import (
  "bytes"
  "encoding/json"
  "fmt"
  "io/ioutil"
  "micrified.com/route"
  "micrified.com/route/blog"
  "micrified.com/route/login"
  "micrified.com/route/logout"
  "micrified.com/service/auth"
  "net/http"
  "os"
  "testing"
)

const (
  Hostname    = "http://localhost:3070"
  BlogRoute   = "/blog"
  LoginRoute  = "/login"
  LogoutRoute = "/logout"
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


// TestBlogImmutableRequests sends a GET request to the /blog endpoint, and 
// retrieves a list of blog headers.
func TestBlogImmutableRequests (t *testing.T) {
  var (
    url  string            = fmt.Sprintf("%s%s", Hostname, BlogRoute)
    body []byte            = []byte{}
    list []blog.BlogHeader = []blog.BlogHeader{}
    err  error             = nil
    res  *http.Response    = nil
  )

  // Fetch blog list; close body after
  if res, err = http.Get(url); nil != err {
    t.Fatalf("GET failed for %s: %v", url, err)
  } else {
    defer res.Body.Close()
  }

  // Read response body
  if body, err = ioutil.ReadAll(res.Body); nil != err {
    t.Fatalf("GET failed for %s: %v", url, err)
  }

  // Unmarshal to type
  if err = json.Unmarshal(body, &list); nil != err {
    t.Fatalf("GET failed for %s: %v", url, err)
  }

}

// Test mutable requests (LOGIN + POST/PUT/DELETE + LOGOUT)

// TestBlogMutableRequests sends the following sequence 
func TestBlogMutableRequests (t *testing.T) {

  var (
    loginURL  string = fmt.Sprintf("%s%s", Hostname, LoginRoute)
    logoutURL string = fmt.Sprintf("%s%s", Hostname, LogoutRoute)
    blogURL   string = fmt.Sprintf("%s%s", Hostname, BlogRoute)
  )

  // Environment variables
  username   := os.Getenv("TEST_USERNAME")
  passphrase := os.Getenv("TEST_PASSPHRASE")

  // LOGIN
  loginFunc := Request[login.SessionCredential, login.LoginCredential]
  loginPost, sessionCredential := login.LoginCredential {
    Username:   username,
    Passphrase: passphrase,
  }, login.SessionCredential{}
  err := loginFunc(loginURL, http.MethodPost, http.StatusOK, loginPost, &sessionCredential)
  if nil != err {
    t.Fatalf("Login POST failed: %v", err)
  }

  // POST
  postFunc := Request[blog.BlogPostResponse, auth.AuthData[blog.BlogPost]]
  blogPost, blogPostResponse := auth.AuthData[blog.BlogPost] {
    Username: username,
    Secret:   sessionCredential.Secret,
    Data: blog.BlogPost {
      Title:    "Nothing Gold Can Stay",
      Subtitle: "Robert Frost",
      Tag:      "Poetry",                     
      Body:     "Nature's first green is gold",
    },
  }, blog.BlogPostResponse{}
  err = postFunc(blogURL, http.MethodPost, http.StatusOK, blogPost, &blogPostResponse)
  if nil != err {
    t.Fatalf("Blog POST failed: %v", err)
  }
  if blogPost.Data.Title != blogPostResponse.Title ||
     blogPost.Data.Subtitle != blogPostResponse.Subtitle ||
     blogPost.Data.Tag != blogPostResponse.Tag ||
     blogPost.Data.Body != blogPostResponse.Body {
    t.Fatalf("POST Response content not as expected!")
  }

  // PUT
  putFunc := Request[blog.BlogPutResponse, auth.AuthData[blog.BlogPut]]
  blogPut, blogPutResponse := auth.AuthData[blog.BlogPut] {
    Username: username,
    Secret:   sessionCredential.Secret,
    Data: blog.BlogPut {
      ID:       blogPostResponse.ID,
      Title:    "Auguries of Innocence",
      Subtitle: "William Blake",
      Tag:      "Poetry",
      Body:     "To see a World in a Grain of Sand",
    },
  }, blog.BlogPutResponse{}
  err = putFunc(blogURL, http.MethodPut, http.StatusOK, blogPut, &blogPutResponse)

  if nil != err {
    t.Fatalf("Blog PUT failed: %v", err)
  }
  if blogPut.Data.ID != blogPutResponse.ID ||
     blogPut.Data.Title != blogPutResponse.Title ||
     blogPut.Data.Subtitle != blogPutResponse.Subtitle ||
     blogPut.Data.Tag != blogPutResponse.Tag ||
     blogPut.Data.Body != blogPutResponse.Body {
    t.Fatalf("PUT Response content not as expected!")
  }

  // DELETE
  delFunc := Request[any, auth.AuthData[blog.BlogDelete]]
  blogDelete := auth.AuthData[blog.BlogDelete] {
    Username: username,
    Secret:   sessionCredential.Secret,
    Data: blog.BlogDelete {
      ID: blogPostResponse.ID,
    },
  }
  err = delFunc(blogURL, http.MethodDelete, http.StatusOK, blogDelete, nil)
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
  err = logoutFunc(logoutURL, http.MethodPost, http.StatusNoContent, logoutPost, nil)
  if nil != err {
    t.Fatalf("POST Response not as expected: %v", err)
  }

}

