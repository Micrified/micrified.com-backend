package main
import (
  "context"
  "fmt"
  "log"
  "micrified.com/internal/user"
  "micrified.com/route"
  "micrified.com/route/blog"
  "micrified.com/route/login"
  "micrified.com/route/logout"
  "micrified.com/service/auth"
  "micrified.com/service/database"
  "net/http"
  "os"
  "time"
)

// handler wraps the given route handler with a context. This context provides
// the user IP address and installs the supplied response timeout. It returns
// a new function to be used as the handler parameter for the http SetHandle
// function. It is imperative that any error and/or status be returned to 
// this function
func handler (c route.Controller) func(http.ResponseWriter, *http.Request) {
  return func (w http.ResponseWriter, rq *http.Request) {
    var (
      elapsed time.Duration = 0
      err     error         = nil
      result  route.Result = route.DefaultResult()
      userIP  string        = ""
    )

    // Wrapper for timed, cancellable request
    doCancellable := func (m route.Method, x context.Context) (time.Duration, error) {
      start, cerr := time.Now(), make(chan error, 1)
      defer close(cerr)
      go func () {
	cerr <-m(c, x, rq.WithContext(x), &result)
      }()
      select {
      case <-x.Done():
	<-cerr
	return time.Since(start), x.Err()
      case err := <-cerr:
	return time.Since(start), err
      }
    }

    // Create context
    x, cancel := context.WithTimeout(context.Background(), c.Timeout())
    defer cancel()

    // Attach user IP; Then handle while measuring
    userIP, err = user.RequestIP(rq)
    if nil == err {
      x = user.ContextWithIP(x, userIP)
      if m := c.Handler(rq.Method); nil != m {
	elapsed, err = doCancellable(m, x)
      } else {
	err = result.Unimplemented()
      }
    }

    // Process result
    if nil != err {
      // TODO: Ensure status code is always != OK (200)? 
      if http.StatusOK == result.Status {
	panic("Cannot return 200 if returning error")
      }
      http.Error(w, err.Error(), result.Status)
    } else {
      w.Header().Set(route.ContentTypeName, result.ContentType)
      w.WriteHeader(result.Status)
      w.Write(result.Buffer.Bytes())
    }

    log.Printf("%s %s %s %d %v\n", userIP, rq.URL.Path, rq.Method, 
      elapsed.Milliseconds(), err)
  }
}



func main() {
  var (
    cfg  Config
    s    route.Service
    err  error
  )

  // Check arguments
  if 2 != len(os.Args) {
    log.Fatalf("usage: %s <config-file>", os.Args[0])
  }

  // Read configuration
  if err = cfg.Read(os.Args[1]); nil != err {
    log.Fatal(err)
  }

  // Setup services
  ds, err := database.NewService(cfg.Database)
  if nil != err {
    log.Fatal(err)
  } else {
    s.Database = &ds
  }
  defer ds.Stop()

  as, err := auth.NewService(cfg.Auth)
  if nil != err {
    log.Fatal(err)
  } else {
    s.Auth = &as
  }

  // Setup route controllers
  blogController, blogListController := blog.NewController(s), blog.NewListController(s)
  loginController  := login.NewController(s)
  logoutController := logout.NewController(s)

  // Install routes
  routes := map[string]func(http.ResponseWriter, *http.Request) {
    blogController.Route()     : handler(&blogController),
    blogListController.Route() : handler(&blogListController),
    loginController.Route()    : handler(&loginController),
    logoutController.Route()   : handler(&logoutController),
  }
  for route, handle := range routes {
    http.HandleFunc(route, handle)
  }

  // Listen and serve
  url := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
  fmt.Printf("Listening at: %s\n", url)
  log.Fatal(http.ListenAndServe(url, nil))
}
