package database

import (
  "context"
  "database/sql"
  "fmt"
  _ "github.com/go-sql-driver/mysql"
)

const (
  MySQL = "mysql"
)


/*\
 *******************************************************************************
 *                            Definition: Functions                            *
 *******************************************************************************
\*/


func DSN (unixSocket, username, password, database string) string {
  return fmt.Sprintf("%s:%s@unix(%s)/%s", username, password, unixSocket,
    database)
}


/*\
 *******************************************************************************
 *                             Definition: Service                             *
 *******************************************************************************
\*/

type Config struct {
  UnixSocket string
  Username   string
  Password   string
  Database   string
}

type Service struct {
  Database string
  Context  context.Context
  DB       *sql.DB
}

func NewService (c Config) (Service, error) {
  var (
    db  *sql.DB = nil
    err error   = nil
    dsn string  = DSN(c.UnixSocket, c.Username, c.Password, c.Database)
  )
  if db, err = sql.Open(MySQL, dsn); nil != err {
    return Service{}, fmt.Errorf("%s could not open: %w", dsn, err)
  }
  if err = db.Ping(); nil != err {
    return Service{}, fmt.Errorf("%s could not be pinged: %w", dsn, err)
  }
  return Service {
    Database: c.Database,
    Context:  context.Background(),
    DB:       db,
  }, nil
}

func (d *Service) Stop () error {
  if err := d.DB.Close(); nil != err {
    return fmt.Errorf("Bad close for DB %s: %w", d.Database, err)
  }
  return nil
}

type TFunc func (sql.Result, *sql.Tx) (sql.Result, error)

func (d *Service) Transaction (fs ...TFunc) (sql.Result, error) {
  var r sql.Result = nil

  // Begin transaction
  t, err := d.DB.BeginTx(d.Context, nil)
  if nil != err {
    return nil, err
  }
  defer t.Rollback() // Has no effect if transaction succeeds

  for _, f := range fs {
    r, err = f(r, t)
    if nil != err {
      return nil, err
    }
  }

  // Commit transaction
  if err = t.Commit(); nil != err {
    return nil, err
  }

  // Return final result
  return r, nil
}

type CFunc func (sql.Result, *sql.Conn) (sql.Result, error)

func (d *Service) Connection (fs ...CFunc) (sql.Result, error) {
  var r sql.Result = nil

  // Begin connection
  c, err := d.DB.Conn(d.Context)
  if nil != err {
    return nil, err
  }
  defer c.Close()

  for _, f := range fs {
    r, err = f(r, c)
    if nil != err {
      return nil, err
    }
  }

  // Return final result
  return r, nil
}

