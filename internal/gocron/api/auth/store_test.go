package auth

import (
	"errors"
	"testing"

	"github.com/dgraph-io/badger/v3"
)

func openMem(t *testing.T) *badger.DB {
	t.Helper()
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLogger(nil))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestStore(t *testing.T) {
	s := NewStore(openMem(t))
	hash, err := HashPassword("password1")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.PutUser(User{Username: "alice", Role: "editor", PasswordHash: hash}); err != nil {
		t.Fatal(err)
	}

	if u, err := s.Authenticate("alice", "password1"); err != nil || u.Role != "editor" {
		t.Fatalf("right password: %+v %v", u, err)
	}
	if _, err := s.Authenticate("alice", "wrong-pass"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong password: %v", err)
	}
	if _, err := s.Authenticate("bob", "password1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("unknown user must look like a wrong password: %v", err)
	}

	users, err := s.ListUsers()
	if err != nil || len(users) != 1 || users[0].PasswordHash != nil {
		t.Fatalf("list must not leak hashes: %+v %v", users, err)
	}

	token, err := s.CreateSession("alice")
	if err != nil {
		t.Fatal(err)
	}
	if name, err := s.SessionUser(token); err != nil || name != "alice" {
		t.Fatalf("session lookup: %q %v", name, err)
	}
	if _, err := s.SessionUser("forged"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("forged token: %v", err)
	}
	if err := s.DeleteSession(token); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SessionUser(token); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted session still valid: %v", err)
	}
}
