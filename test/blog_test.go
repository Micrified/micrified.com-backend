package main

import (
  "bytes"
  "encoding/json"
  "fmt"
  "io/ioutil"
  "log"
  "micrified.com/route"
  "micrified.com/route/blog"
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
  LoginURL string  = os.Getenv("TEST_HOSTNAME") + "/" + login.RouteName
  LogoutURL string = os.Getenv("TEST_HOSTNAME") + "/" + logout.RouteName
  BlogURL string   = os.Getenv("TEST_HOSTNAME") + "/" + blog.RouteName
  BlogListURL string   = os.Getenv("TEST_HOSTNAME") + "/" + blog.RouteListName
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


// TestBlogList sends a GET request to the bloglist endpoint and 
// retrieves a list of blog headers.
func TestBlogList (t *testing.T) {
  var (
    body []byte            = []byte{}
    list []blog.BlogHeader = []blog.BlogHeader{}
    err  error             = nil
    res  *http.Response    = nil
  )

  // Fetch blog list; close body after
  if res, err = http.Get(BlogListURL); nil != err {
    t.Fatalf("GET failed for %s: %v", BlogListURL, err)
  } else {
    defer res.Body.Close()
  }

  // Read response body
  if body, err = ioutil.ReadAll(res.Body); nil != err {
    t.Fatalf("GET failed for %s: %v", BlogListURL, err)
  }

  // Unmarshal to type
  if err = json.Unmarshal(body, &list); nil != err {
    t.Fatalf("GET failed for %s: %v", BlogListURL, err)
  }

}

// TestBlog performs a login, then tests the following REST operations
// of blog: LOGIN + POST + GET + PUT + GET + DELETE + LOGOUT
func TestBlog (t *testing.T) {

  // Environment variables
  username   := os.Getenv("TEST_USERNAME")
  passphrase := os.Getenv("TEST_PASSPHRASE")

  // getBlog performs a GET request to the blog endpoint
  getBlog := func(blog_id string, blogResponse *blog.BlogResponse) error {
    req, err := http.NewRequest(http.MethodGet, BlogURL, nil)
    if nil != err {
      return err
    }
    q := req.URL.Query()
    q.Set("id", blog_id)
    req.URL.RawQuery = q.Encode()
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
    return json.Unmarshal(body, blogResponse)
  }

  // LOGIN
  loginFunc := Request[login.SessionCredential, login.LoginCredential]
  loginPost, sessionCredential := login.LoginCredential {
    Username:   username,
    Passphrase: passphrase,
  }, login.SessionCredential{}
  err := loginFunc(LoginURL, http.MethodPost, http.StatusOK, loginPost, &sessionCredential)
  if nil != err {
    t.Fatalf("Login POST failed: %v", err)
  }

  // POST
  postFunc := Request[blog.PostResponse, auth.Frame[blog.Post]]
  blogPost, blogPostResponse := auth.Frame[blog.Post] {
    Username: username,
    Secret:   sessionCredential.Secret,
    Data: blog.Post {
      Title:    "Nothing Gold Can Stay",
      Subtitle: "Robert Frost",
      Body:     "Nature's first green is gold",
    },
  }, blog.PostResponse{}
  err = postFunc(BlogURL, http.MethodPost, http.StatusOK, blogPost, &blogPostResponse)
  if nil != err {
    t.Fatalf("Blog POST failed: %v", err)
  }
  if blogPost.Data.Title != blogPostResponse.Title ||
     blogPost.Data.Subtitle != blogPostResponse.Subtitle ||
     blogPost.Data.Body != blogPostResponse.Body {
    t.Fatalf("POST Response content not as expected!")
  }

  // GET
  blogResponse := blog.BlogResponse{}
  err = getBlog(blogPostResponse.ID, &blogResponse)
  if nil != err {
    t.Fatalf("Blog GET failed: %v", err)
  }
  if blogResponse.Title    != blogPostResponse.Title    ||
     blogResponse.Subtitle != blogPostResponse.Subtitle ||
     blogResponse.Body     != blogPostResponse.Body     ||
     blogResponse.Created  != blogPostResponse.Created  ||
     blogResponse.Updated  != blogPostResponse.Updated {
    t.Fatalf("GET blog content not as expected!")
  }

  // PUT
  putFunc := Request[blog.PutResponse, auth.Frame[blog.Put]]
  blogPut, blogPutResponse := auth.Frame[blog.Put] {
    Username: username,
    Secret:   sessionCredential.Secret,
    Data: blog.Put {
      ID:       blogPostResponse.ID,
      Title:    "Auguries of Innocence",
      Subtitle: "William Blake",
      Body:     "To see a World in a Grain of Sand",
    },
  }, blog.PutResponse{}
  err = putFunc(BlogURL, http.MethodPut, http.StatusOK, blogPut, &blogPutResponse)

  if nil != err {
    t.Fatalf("Blog PUT failed: %v", err)
  }
  if blogPut.Data.ID != blogPutResponse.ID ||
     blogPut.Data.Title != blogPutResponse.Title ||
     blogPut.Data.Subtitle != blogPutResponse.Subtitle ||
     blogPut.Data.Body != blogPutResponse.Body {
    t.Fatalf("PUT Response content not as expected!")
  }

  // GET
  err = getBlog(blogPostResponse.ID, &blogResponse)
  if nil != err {
    t.Fatalf("Blog GET failed: %v", err)
  }
  if blogResponse.Title    != blogPutResponse.Title    ||
     blogResponse.Subtitle != blogPutResponse.Subtitle ||
     blogResponse.Body     != blogPutResponse.Body {
    t.Fatalf("GET blog content not as expected!")
  }

  // DELETE
  delFunc := Request[any, auth.Frame[blog.Delete]]
  blogDelete := auth.Frame[blog.Delete] {
    Username: username,
    Secret:   sessionCredential.Secret,
    Data: blog.Delete {
      ID: blogPostResponse.ID,
    },
  }
  err = delFunc(BlogURL, http.MethodDelete, http.StatusOK, blogDelete, nil)
  if nil != err {
    t.Fatalf("DELETE Response not as expected: %v", err)
  }

  // LOGOUT
  logoutFunc := Request[any, auth.Frame[logout.LogoutCredential]]
  logoutPost := auth.Frame[logout.LogoutCredential]{
    Username: username,
    Secret:   sessionCredential.Secret,
    Data:     logout.LogoutCredential{},
  }
  err = logoutFunc(LogoutURL, http.MethodPost, http.StatusNoContent, logoutPost, nil)
  if nil != err {
    t.Fatalf("POST Response not as expected: %v", err)
  }

}

