package auth

import (
  "crypto/rand"
  "math"
  "sync"
  "time"
)

const (
  MinSessionPeriod = 1 * time.Second
  MaxSessionPeriod = 1 * time.Hour
)


/*\
 *******************************************************************************
 *                         Definitions: Generic types                          *
 *******************************************************************************
\*/


type SyncMap [T comparable, U any] struct {
  m     map[T]U
  mutex sync.Mutex
}

func (s *SyncMap[T,U]) Get(t T) (U, bool) {
  s.mutex.Lock()
  defer s.mutex.Unlock()
  u, ok := s.m[t]
  return u, ok
}

func (s *SyncMap[T,U]) Put (t T, u U) {
  s.mutex.Lock()
  defer s.mutex.Unlock()
  s.m[t] = u
}

func (s *SyncMap[T,U]) Delete (t T) {
  s.mutex.Lock()
  defer s.mutex.Unlock()
  delete(s.m, t)
}

func NewSyncMap [T comparable, U any] () SyncMap[T,U] {
  return SyncMap[T,U] {
    m: make(map[T]U),
    mutex: sync.Mutex{},
  }
}

type Frame [T any] struct {
  Username string `json:"username"`
  Secret   string `json:"secret"`
  Data     T      `json:"data"`
}


/*\
 *******************************************************************************
 *                             Definition: Penalty                             *
 *******************************************************************************
\*/


type Penalty struct {
  Deadline  time.Time
  Count     int
}

// PenaltyFunc: Applies penalty function and returns the duration in seconds
func penaltyFunc(i, base, factor, retry int) int {
  return base * int(math.Pow(float64(factor), float64(i - retry)))
}

// Refresh: Applies the piecewise exponential penalty function:
// f(x) = | x < retry : 0,
//        | x >= retry : base * factor^(x - retry)
func (p *Penalty) Refresh (c *Config) Penalty {
  duration, increment := 0, 0
  if p.Count >= c.Retry {
    duration = penaltyFunc(p.Count, c.Base, c.Factor, c.Retry)
  }
  if duration < c.Limit {
    increment = 1
  }
  return Penalty {
    Deadline: time.Now().UTC().Add(time.Duration(duration) * time.Second),
    Count:    p.Count + increment,
  }
}

// NewPenalty: Returns new penalty instantiated with hardcoded start duration
func NewPenalty (c *Config) Penalty {
  duration := 0
  if c.Retry == 0 {
    duration = penaltyFunc(0, c.Base, c.Factor, c.Retry)
  }
  return Penalty {
    Deadline: time.Now().UTC().Add(time.Duration(duration) * time.Second),
    Count:    0,
  }
}


/*\
 *******************************************************************************
 *                             Definition: Session                             *
 *******************************************************************************
\*/


type Session struct {
  Expiration time.Time
  Period     time.Duration
  IP         string
  Secret     Hash
}

// Expired: Returns true if the given session is invalid 
func (s *Session) Expired () bool {
  t := time.Now().UTC()
  return t.After(s.Expiration)
}

// Renew: Returns a new session with expiration: current time + duration
func (s *Session) Renew () Session {
  return Session {
    Expiration: time.Now().UTC().Add(s.Period),
    Period:     s.Period,
    IP:         s.IP,
    Secret:     s.Secret,
  }
}

// NewSession: Returns new session for given IP and duration
func NewSession (ip string, period time.Duration) (Session, error) {
  var (
    b      []byte = make([]byte, HashSize)
    secret Hash   = Hash{}
  )

  if _, err := rand.Read(b); nil != err {
    return Session{}, err
  }

  // Copy random buffer into hash
  copy(secret[:], b[:HashSize])

  return Session {
    Expiration: time.Now().UTC().Add(period),
    Period:     period,
    IP:         ip,
    Secret:     secret,
  }, nil
}





















































