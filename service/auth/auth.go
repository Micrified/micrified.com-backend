// Package auth provides facilities for hashing and management of secrets.
// The generation of new secrets, and the conversion of digests to secrets
// are the primary functions provided
//
// For generating salts, auth relies on the crytographically secure
// pseudorandom generator in package rand: https://pkg.go.dev/crypto/rand
// For generating secure hashes, rand is used in conjunction with 
// SHAKE-256. This is implemented in the sha3 package:
// https://pkg.go.dev/golang.org/x/crypto/sha3

package auth

import (
  "bytes"
  "crypto/rand"
  "encoding/hex"
  "fmt"
  "golang.org/x/crypto/sha3"
  "strconv"
  "sync"
  "time"
)

const (
  HashSize = 64
)


/*\
 *******************************************************************************
 *                     Definitions: Crytographic Functions                     *
 *******************************************************************************
\*/


// Defines: Standard hash type
type Hash [HashSize]byte

// Returns the hash as a hex encoded string
func (h *Hash) HexString() string {
  return hex.EncodeToString(ToByteSlice(*h))
}

// Returns whether two hashes are equal
func (h *Hash) Equal(other *Hash) bool {
  var i int = 0
  for i < HashSize && h[i] == other[i] {
    i++
  }
  return (i == HashSize)
}

// Returns a hash as a byte sequence
func ToByteSlice (h Hash) []byte {
  var b []byte = make([]byte, HashSize)
  copy(b[:], h[:HashSize])
  return b
}


// NewSecret: Returns a new cryptographically secure (secret,salt) hash pair
func NewSecret (digest string) (secret, salt Hash, err error) {
  var b []byte = make([]byte, HashSize)
  if _, err = rand.Read(b); nil != err {
    return
  }

  // Copy random buffer into salt
  copy(salt[:], b[:HashSize])

  // Hash digest into the salt to make the secret
  sha3.ShakeSum256(b, []byte(digest))
  copy(secret[:], b[:HashSize])
  return
}

// ToSecret: Derives a crytographically secure secret given a digest and salt
func ToSecret (digest string, salt Hash) (secret Hash) {
  var b []byte = make([]byte, HashSize)

  // Convert the salt to []byte
  copy(b[:HashSize], salt[:])

  // Hash digest into the salt to make the secret
  sha3.ShakeSum256(b, []byte(digest))
  copy(secret[:], b[:HashSize])
  return
}

/*\
 *******************************************************************************
 *                           Definitions: Service                              *
 *******************************************************************************
\*/


type Config struct {
  Base   int
  Factor int
  Limit  int
  Retry  int
}

type Service struct {
  config     Config
  penalties  SyncMap[string, Penalty]
  sessions   SyncMap[string, Session]
  mutex      sync.Mutex
}

func NewService (c Config) (Service, error) {
  // TODO: Provide penalty expression as regex string 
  if c.Retry < 0 || c.Base < 1 {
    return Service{}, fmt.Errorf("Unmet condition: Retry >= 0, Base >= 1")
  }
  if c.Limit < c.Base {
    return Service{}, fmt.Errorf("Unmet condition: Base <= Limit")
  }
  if c.Factor < 1 {
    return Service{}, fmt.Errorf("Unmet condition: 1 < Factor")
  }
  return Service {
    config:     c,
    penalties:  NewSyncMap[string, Penalty](),
    sessions:   NewSyncMap[string, Session](),
    mutex:      sync.Mutex{},
  }, nil
}

// Penalised returns true if the given IP has an assigned penalty
// The method is thread safe
func (s *Service) Penalised (ip string) bool {
  if penalty, ok := s.penalties.Get(ip); ok {
    return time.Now().Before(penalty.Deadline)
  }
  return false
}

// Penalise installs or refreshes a penalty for the given IP
func (s *Service) Penalise (ip string) {
  if penalty, ok := s.penalties.Get(ip); ok {
    updated := penalty.Refresh(&s.config)
    s.penalties.Put(ip, updated)
  } else {
    s.penalties.Put(ip, NewPenalty(&s.config))
  }
}

// NoPenalty removes any registered penalty for the given IP
func (s *Service) NoPenalty (ip string) {
  s.penalties.Delete(ip)
}

// Compare returns true if hash(digest, salt) == hash
func Compare (digest string, salt, hash []byte) bool {
  b := make([]byte, HashSize)

  // Copy salt into buffer
  copy(b[:], salt[:HashSize])

  // Create hash = ShakeSum256(digest, salt) into b
  sha3.ShakeSum256(b, []byte(digest))

  // Compare buffer
  return bytes.Equal(b, hash)
}

// Authentication signature: Returns true,nil if successful
type AuthFunc func () (bool, error)

// Authenticate executes given authentication function in thread safe context.
// If the authentication function returns (true, nil), then a new session is
// created and returned by value. Otherwise, a default session is returned
// and the returned values of the authentication function propagated back
func (s *Service) Authenticate (ip, username, period string, f AuthFunc) (Session, bool, error) {
  var (
    z   Session = Session{}
    ok  bool    = false
    err error   = nil
  )

  // Secure mutual exclusion for duration of authentication
  s.mutex.Lock()
  defer s.mutex.Unlock()

  // Case: error during auth or bad credentials
  if ok, err = f(); nil != err || !ok {
    return z, ok, err
  }

  // Parse session period; apply limits
  t := MaxSessionPeriod
  if value, err := strconv.Atoi(period); nil == err {
    t = max(min(time.Duration(value) * time.Second, MaxSessionPeriod), 
      MinSessionPeriod)
  }

  // Create session; register if no error
  if z, err = NewSession(ip, t); nil == err {
    s.sessions.Put(username, z)
  }

  return z, ok, err
}

// Deauthenticate checks whether a session exists for the given username
// and removes the associated session, if so. It is thread safe.
// Note: This function assumes the invoking request is authenticated,
// but does not verify. Ensure Authenticate has been performed first for
// the request!
func (s *Service) Deauthenticate (username string) error {

  // No mutex holding needed (single access to delete on atomic map)

  // Delete (assumed username exists);
  s.sessions.Delete(username)

  return nil
}

// Authorized checks the provided session secret and checks whether it exists
// and not expired. It is thread-safe
func (s *Service) Authorized (ip, username, secret string) error {
  var z Session

  // Grab mutex and lock (need continuous mutual exclusion until renew)
  s.mutex.Lock()
  defer s.mutex.Unlock()

  // Case: There is no session for the given username
  z, ok := s.sessions.Get(username)
  if !ok {
    return fmt.Errorf("No session for username \"%s\"", username)
  }

  // Case: The secret doesn't match that of the session
  if secret != z.Secret.HexString() {
    return fmt.Errorf("Session secret mismatch")
  }
  
  // Case: The IP doesn't match that used to create the session
  if ip != z.IP {
    return fmt.Errorf("Session IP mismatch")
  }

  // Case: The secret has expired
  if z.Expired() {
    return fmt.Errorf("Session expired")
  }

  // Renew session validity
  s.sessions.Put(username, z.Renew())

  return nil
}
