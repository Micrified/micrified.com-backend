package database

import (
  "database/sql/driver"
  "encoding/json"
  "fmt"
  "time"
)

/*\
 *******************************************************************************
 *                              Definition: Types                              *
 *******************************************************************************
\*/

// RFC3339 
type DateTime time.Time

// Stringer
func (t DateTime) String () string {
  return time.Time(t).Format(time.RFC3339)
}

// Marshal/Unmarshal JSON
func (t DateTime) MarshalJSON () ([]byte, error) {
  return json.Marshal(time.Time(t).Format(time.RFC3339))
}
func (p *DateTime) UnmarshalJSON (data []byte) error {
  var (
    v string
    t time.Time
  )
  if err := json.Unmarshal(data, &v); nil != err {
    return err
  }
  t, err := time.Parse(time.RFC3339, v)
  if nil != err {
    return err
  }
  *p = DateTime(t)
  return nil
}

// Valuer / Scanner
func (t DateTime) Value () (driver.Value, error) {
  return time.Time(t), nil
}
func (p *DateTime) Scan (src any) error {
  if nil == src {
    return fmt.Errorf("No lossless conversion from nil")
  }
  t, ok := src.(time.Time)
  if !ok {
    return fmt.Errorf("Invalid type conversion")
  }
  *p = DateTime(t)
  return nil
}

