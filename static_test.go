package main

import (
  "encoding/json"
  "io/ioutil"
  "log"
  "micrified.com/route/static"
  "net/http"
  "os"
  "testing"
)



/*\
 *******************************************************************************
 *                                    Tests                                    *
 *******************************************************************************
\*/


var (
  StaticURL string  = os.Getenv("TEST_HOSTNAME") + "/" + static.RouteName
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


// TestStaticIndex sends a GET request to the static endpoint to fetch the
// canonical "index.html" resource
func TestStaticIndex (t *testing.T) {
  var (
    body []byte              = []byte{}
    index static.GetResponse = static.GetResponse{}
    err  error               = nil
    res  *http.Response      = nil
  )

  // Compose the index URL
  indexURL := StaticURL + "/index"

  // Fetch index
  if res, err = http.Get(indexURL); nil != err {
    t.Fatalf("GET failed for %s: %v", indexURL, err)
  } else {
    defer res.Body.Close()
  }

  // Read response body
  if body, err = ioutil.ReadAll(res.Body); nil != err {
    t.Fatalf("GET failed for %s: %v", indexURL, err)
  }

  // Unmarshal to type
  if err = json.Unmarshal(body, &index); nil != err {
    t.Fatalf("GET failed for %s: %v", indexURL, err)
  }

  log.Printf("Got index: %+v\n", index)
}

