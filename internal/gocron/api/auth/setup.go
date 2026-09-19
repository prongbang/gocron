package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v3"
)

// Setup builds the enforcer and store, and creates the first admin from the given credentials
// when no user exists yet. Later calls never touch existing users.
func Setup(db *badger.DB, adminUser, adminPassword string) (*Auth, error) {
	e, err := NewEnforcer()
	if err != nil {
		return nil, err
	}
	s := NewStore(db)
	users, err := s.ListUsers()
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		if !usernameRe.MatchString(adminUser) || !validPassword(adminPassword) {
			return nil, errors.New("GOCRON_AUTH=true needs GOCRON_ADMIN_USER (3-32 chars of a-z 0-9 _ . -) and GOCRON_ADMIN_PASSWORD (8-72 chars) to create the first admin")
		}
		hash, err := HashPassword(adminPassword)
		if err != nil {
			return nil, err
		}
		if err := s.PutUser(User{Username: adminUser, Role: "admin", PasswordHash: hash, CreatedAt: time.Now()}); err != nil {
			return nil, err
		}
		fmt.Println("[INFO] Created first admin:", adminUser)
	}
	return &Auth{Store: s, Enforcer: e}, nil
}
