package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"time"

	"github.com/dgraph-io/badger/v3"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Username     string    `json:"username"`
	Role         string    `json:"role"`
	PasswordHash []byte    `json:"password_hash,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

var (
	ErrNotFound           = errors.New("not found")
	ErrInvalidCredentials = errors.New("invalid username or password")
)

// ponytail: fixed 24h sessions, make configurable if someone needs a different lifetime.
const sessionTTL = 24 * time.Hour

var usernameRe = regexp.MustCompile(`^[a-z0-9_.-]{3,32}$`)

// bcrypt ignores everything after 72 bytes, so longer passwords would be silently truncated.
func validPassword(p string) bool { return len(p) >= 8 && len(p) <= 72 }

// Store keeps users ("user:<name>") and sessions ("session:<sha256>") in gocron's badger DB.
type Store struct{ db *badger.DB }

func NewStore(db *badger.DB) *Store { return &Store{db: db} }

func HashPassword(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}

func userKey(name string) []byte { return []byte("user:" + name) }

func (s *Store) PutUser(u User) error {
	v, err := json.Marshal(u)
	if err != nil {
		return err
	}
	return s.db.Update(func(txn *badger.Txn) error { return txn.Set(userKey(u.Username), v) })
}

func (s *Store) GetUser(name string) (User, error) {
	var u User
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(userKey(name))
		if errors.Is(err, badger.ErrKeyNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		return item.Value(func(v []byte) error { return json.Unmarshal(v, &u) })
	})
	return u, err
}

// ListUsers returns every user without password hashes.
func (s *Store) ListUsers() ([]User, error) {
	users := []User{}
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte("user:")
		itr := txn.NewIterator(opts)
		defer itr.Close()
		for itr.Rewind(); itr.Valid(); itr.Next() {
			var u User
			if err := itr.Item().Value(func(v []byte) error { return json.Unmarshal(v, &u) }); err != nil {
				return err
			}
			u.PasswordHash = nil
			users = append(users, u)
		}
		return nil
	})
	return users, err
}

func (s *Store) DeleteUser(name string) error {
	return s.db.Update(func(txn *badger.Txn) error { return txn.Delete(userKey(name)) })
}

// dummyHash makes a login for an unknown user as slow as a wrong password,
// so response time does not reveal which usernames exist.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("gocron-dummy"), bcrypt.DefaultCost)

func (s *Store) Authenticate(username, password string) (User, error) {
	u, err := s.GetUser(username)
	hash := u.PasswordHash
	if err != nil {
		hash = dummyHash
	}
	if bcrypt.CompareHashAndPassword(hash, []byte(password)) != nil || err != nil {
		return User{}, ErrInvalidCredentials
	}
	return u, nil
}

// Only a hash of each token is stored, so a copy of the DB does not hand out live sessions.
func sessionKey(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return []byte("session:" + hex.EncodeToString(sum[:]))
}

func (s *Store) CreateSession(username string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	err := s.db.Update(func(txn *badger.Txn) error {
		return txn.SetEntry(badger.NewEntry(sessionKey(token), []byte(username)).WithTTL(sessionTTL))
	})
	return token, err
}

func (s *Store) SessionUser(token string) (string, error) {
	var name string
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(sessionKey(token))
		if errors.Is(err, badger.ErrKeyNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		v, err := item.ValueCopy(nil)
		name = string(v)
		return err
	})
	return name, err
}

func (s *Store) DeleteSession(token string) error {
	return s.db.Update(func(txn *badger.Txn) error { return txn.Delete(sessionKey(token)) })
}
