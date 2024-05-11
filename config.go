package main

import (
  "encoding/json"
  "fmt"
  "micrified.com/service/auth"
  "micrified.com/service/database"
  "os"
)

type Config struct {
  Auth         auth.Config
  Database     database.Config
  Host         string
  Port         string
}

func (c *Config) Read (filepath string) error {
  f, err := os.Open(filepath)
  if nil != err {
    return fmt.Errorf("Couldn't open configuration %s: %w", filepath, err)
  } else {
    defer f.Close()
  }
  parser := json.NewDecoder(f)
  return parser.Decode(c)
}
