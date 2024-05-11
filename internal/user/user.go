package user

import (
  "net"
  "net/http"
  "fmt"
  "context"
)

const (
  UserIPKey = 0
)

func RequestIP(r *http.Request) (string, error) {
  ip, _, err := net.SplitHostPort(r.RemoteAddr)
  if nil != err {
    return "", fmt.Errorf("Host %q is not of form 'host:port' or similar")
  }
  return ip, nil
}

func ContextWithIP(c context.Context, ip string) context.Context {
  return context.WithValue(c, UserIPKey, ip)
}
