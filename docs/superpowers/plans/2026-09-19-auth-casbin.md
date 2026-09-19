# Authentication + Casbin Authorization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Require sign-in for the gocron API and dashboard, and let Casbin decide what each signed-in user's role may do.

**Architecture:** Casbin only answers "may role X call METHOD /path?"; it does not identify anyone. So authentication is ours: users (bcrypt) and opaque session tokens live in the existing badger DB, sessions expire via badger TTL. One fiber middleware resolves the session (cookie for the dashboard, `Authorization: Bearer` for scripts), then asks a Casbin RESTful model (`keyMatch2` + `regexMatch`) with an embedded policy. The whole feature is off unless `GOCRON_AUTH=true`, so current deployments keep working.

**Tech Stack:** Go 1.21 module, fiber v2.52.5, badger v3, `github.com/casbin/casbin/v2` v2.135.0, `golang.org/x/crypto/bcrypt` v0.31.0; SvelteKit 2 + Svelte 5 runes + shadcn-svelte, bun.

**Spec:** This document. The design decisions below are the spec. Confirm them with the owner before Task 1.

## Design decisions (confirm before starting)

| # | Decision | Why |
|---|---|---|
| D1 | Off by default; `GOCRON_AUTH=true` turns it on | Existing users run an open API; flipping it on silently would break their scripts |
| D2 | First admin comes from `GOCRON_ADMIN_USER` / `GOCRON_ADMIN_PASSWORD`, only when no user exists | Otherwise nobody can ever sign in; later restarts never reset the password |
| D3 | Opaque random session tokens (32 bytes), SHA-256 stored in badger with a 24h TTL; no JWT | Logout and user deletion take effect at once, no signing key to manage, no new dependency |
| D4 | Token via `HttpOnly; SameSite=Strict` cookie for the dashboard, or `Authorization: Bearer` for scripts | Same-origin dashboard gets CSRF protection from SameSite; scripts get a header |
| D5 | Three global roles `admin`, `editor`, `viewer`, one per user, stored on the user record | Enough for "who may change jobs"; project-scoped roles are a follow-up (see end) |
| D6 | Casbin RESTful model, policy embedded in the binary (`policy.csv`) | Permissions are data, not `if` statements; changing them needs a rebuild (acceptable for 3 fixed roles) |
| D7 | Dashboard static files stay public; only `/v1/*` is protected | The SPA must load to show the login page; it contains no data |
| D8 | Login limited to 10 attempts per minute per IP (fiber `limiter`) | Cheap brute-force brake |

Role permissions (the policy this plan ships):

| Role | `GET /v1/scheduler` `GET /v1/history` | `POST /v1/scheduler` `POST /v1/scheduler/stop` | `/v1/users*` |
|---|---|---|---|
| viewer | yes | no | no |
| editor | yes | yes | no |
| admin | yes | yes | yes |

## Global Constraints

- Keep `go 1.21.0` in `go.mod`. `golang.org/x/crypto@latest` needs Go 1.26, so pin `v0.31.0`. After every `go get`, run `git diff go.mod`; if the `go` line changed, revert it and pin an older version of the module that caused it.
- No new npm dependencies. Use the shadcn-svelte components already in `web/src/lib/components/ui/` and follow `.agents/skills/shadcn-svelte/SKILL.md`.
- Badger keyspace: jobs are bare 32-hex UUIDs; everything else is `<kind>:…` (`history:`, `user:`, `session:`). Never add an un-prefixed key.
- Usernames: `^[a-z0-9_.-]{3,32}$`. Passwords: 8–72 bytes (bcrypt ignores bytes after 72).
- Commits: Conventional Commits, English, no AI/assistant references.
- Tests: `go test ./...`, `go vet ./...` and `cd web && bun run check` must pass at the end of every task that touches them.

## File Structure

| File | Responsibility |
|---|---|
| `internal/gocron/api/scheduler/repository.go` (modify) | Job listing ignores every namespaced key, not only `history:` |
| `internal/gocron/api/scheduler/keys_test.go` (create) | Regression test for the keyspace rule |
| `internal/gocron/api/auth/model.conf` (create) | Casbin RESTful model |
| `internal/gocron/api/auth/policy.csv` (create) | Role → path/method permissions |
| `internal/gocron/api/auth/enforcer.go` (create) | Builds the enforcer from the embedded files; `Roles` |
| `internal/gocron/api/auth/store.go` (create) | Users + sessions in badger, bcrypt, `Authenticate` |
| `internal/gocron/api/auth/http.go` (create) | Middleware, login/logout/me, user CRUD handlers, `Routes` |
| `internal/gocron/api/auth/setup.go` (create) | `Setup`: enforcer + store + first-admin bootstrap |
| `internal/gocron/api/auth/*_test.go` (create) | Policy table, store, HTTP flow, setup |
| `pkg/core/response.go` (modify) | `Unauthorized`, `Forbidden` helpers |
| `internal/gocron/api/api.go`, `provider.go` (modify) | Read `GOCRON_AUTH`, register auth before the routes it protects |
| `web/src/lib/api.ts` (modify) | 401 → login redirect; auth + users client calls |
| `web/src/lib/session.svelte.ts` (create) | Who is signed in; `canWriteJobs()` |
| `web/src/routes/login/+page.svelte` (create) | Sign-in page |
| `web/src/routes/+layout.svelte` (modify) | User badge, sign out, Users link, no nav on /login |
| `web/src/routes/+page.svelte`, `web/src/lib/components/job-table.svelte` (modify) | Hide New job / Stop for viewers |
| `web/src/lib/components/create-user-dialog.svelte`, `web/src/routes/users/+page.svelte` (create) | Admin user management |
| `README.md` (modify) | Env vars, roles, curl example |

---

### Task 1: Reserve the badger keyspace for jobs

`GetConfigAll` unmarshals every key that is not `history:…` as a job. `user:` and `session:` records added later would show up as ghost jobs. The fix goes first so nothing else can trip over it.

**Files:**
- Modify: `internal/gocron/api/scheduler/repository.go` (the `HasPrefix(item.Key(), historyPrefix)` check inside `GetConfigAll`)
- Test: `internal/gocron/api/scheduler/keys_test.go`

**Interfaces:**
- Consumes: `memDrivers` from `history_test.go` (same package)
- Produces: rule "job keys contain no `:`", relied on by Task 3's `user:` / `session:` keys

- [ ] **Step 1: Write the failing test**

```go
package scheduler

import (
	"testing"

	"github.com/dgraph-io/badger/v3"
)

func TestGetConfigAllIgnoresNamespacedKeys(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLogger(nil))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewRepository(memDrivers{db})

	if err := repo.Add("5c964cf984484051a5ce10536c039a5a", CreateScheduler{Job: "5c964cf984484051a5ce10536c039a5a"}); err != nil {
		t.Fatal(err)
	}
	err = db.Update(func(txn *badger.Txn) error {
		for _, k := range []string{"user:alice", "session:abc"} {
			if err := txn.Set([]byte(k), []byte(`{"job":"not-a-job"}`)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if got := repo.GetConfigAll(); len(got) != 1 {
		t.Fatalf("want only the real job, got %+v", got)
	}
}
```

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./internal/gocron/api/scheduler/ -run TestGetConfigAllIgnoresNamespacedKeys -v`
Expected: FAIL, `want only the real job, got [...3 items...]`

- [ ] **Step 3: Implement**

In `repository.go`, add below the imports:

```go
// Job keys are bare UUIDs; every other record in the shared badger DB is namespaced
// as "<kind>:…" (history:, user:, session:), so anything with a colon is not a job.
func isJobKey(k []byte) bool { return !bytes.Contains(k, []byte(":")) }
```

and in `GetConfigAll` replace

```go
			if bytes.HasPrefix(item.Key(), historyPrefix) {
				continue
			}
```

with

```go
			if !isJobKey(item.Key()) {
				continue
			}
```

- [ ] **Step 4: Run the package tests**

Run: `go test ./internal/gocron/api/scheduler/ -v`
Expected: PASS (`TestHistory` and the new test)

- [ ] **Step 5: Commit**

```bash
git add internal/gocron/api/scheduler/repository.go internal/gocron/api/scheduler/keys_test.go
git commit -m "fix(api): ignore all namespaced badger keys when listing jobs"
```

---

### Task 2: Casbin policy

**Files:**
- Create: `internal/gocron/api/auth/model.conf`, `internal/gocron/api/auth/policy.csv`, `internal/gocron/api/auth/enforcer.go`
- Test: `internal/gocron/api/auth/enforcer_test.go`

**Interfaces:**
- Produces: `func NewEnforcer() (*casbin.Enforcer, error)`, `var Roles = []string{"admin", "editor", "viewer"}`. Callers pass `(role, lowercasePathWithoutTrailingSlash, method)` to `Enforce`.

- [ ] **Step 1: Add the dependencies**

```bash
go get github.com/casbin/casbin/v2@v2.135.0 golang.org/x/crypto@v0.31.0
git diff go.mod
```

Expected: only `require` lines change; the `go 1.21.0` line is untouched (see Global Constraints if it moved).

- [ ] **Step 2: Write the failing test**

```go
package auth

import "testing"

func TestPolicy(t *testing.T) {
	e, err := NewEnforcer()
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		role, path, method string
		want               bool
	}{
		{"viewer", "/v1/scheduler", "GET", true},
		{"viewer", "/v1/history", "GET", true},
		{"viewer", "/v1/scheduler", "POST", false},
		{"viewer", "/v1/scheduler/stop", "POST", false},
		{"viewer", "/v1/users", "GET", false},
		{"editor", "/v1/scheduler", "POST", true},
		{"editor", "/v1/scheduler/stop", "POST", true},
		{"editor", "/v1/users", "GET", false},
		{"admin", "/v1/users", "POST", true},
		{"admin", "/v1/users/bob", "DELETE", true},
		{"admin", "/v1/scheduler", "PATCH", false}, // unknown methods stay closed
		{"nobody", "/v1/scheduler", "GET", false},
	}
	for _, c := range cases {
		got, err := e.Enforce(c.role, c.path, c.method)
		if err != nil || got != c.want {
			t.Errorf("%s %s %s = %v (err %v), want %v", c.role, c.method, c.path, got, err, c.want)
		}
	}
}
```

- [ ] **Step 3: Run it and watch it fail**

Run: `go test ./internal/gocron/api/auth/ -run TestPolicy -v`
Expected: FAIL, `undefined: NewEnforcer`

- [ ] **Step 4: Implement**

`model.conf`:

```ini
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = r.sub == p.sub && keyMatch2(r.obj, p.obj) && regexMatch(r.act, p.act)
```

`policy.csv` (anchored regexes; `regexMatch` is otherwise a substring match):

```csv
p, admin, /v1/*, ^(GET|POST|PUT|DELETE)$
p, editor, /v1/scheduler, ^(GET|POST)$
p, editor, /v1/scheduler/stop, ^POST$
p, editor, /v1/history, ^GET$
p, viewer, /v1/scheduler, ^GET$
p, viewer, /v1/history, ^GET$
```

`enforcer.go`:

```go
// Package auth adds sign-in (users + sessions in badger) and Casbin role checks to the gocron API.
package auth

import (
	_ "embed"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	stringadapter "github.com/casbin/casbin/v2/persist/string-adapter"
)

//go:embed model.conf
var modelConf string

//go:embed policy.csv
var policyCSV string

// Roles a user can have; what each may do lives in policy.csv.
var Roles = []string{"admin", "editor", "viewer"}

func NewEnforcer() (*casbin.Enforcer, error) {
	m, err := model.NewModelFromString(modelConf)
	if err != nil {
		return nil, err
	}
	return casbin.NewEnforcer(m, stringadapter.NewAdapter(policyCSV))
}
```

- [ ] **Step 5: Run the test**

Run: `go test ./internal/gocron/api/auth/ -run TestPolicy -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum internal/gocron/api/auth/model.conf internal/gocron/api/auth/policy.csv internal/gocron/api/auth/enforcer.go internal/gocron/api/auth/enforcer_test.go
git commit -m "feat(auth): add casbin role policy for the api"
```

---

### Task 3: Users and sessions store

**Files:**
- Create: `internal/gocron/api/auth/store.go`
- Test: `internal/gocron/api/auth/store_test.go`

**Interfaces:**
- Produces:
  - `type User struct { Username string; Role string; PasswordHash []byte; CreatedAt time.Time }` (JSON: `username`, `role`, `password_hash,omitempty`, `created_at`)
  - `var ErrNotFound, ErrInvalidCredentials error`; `const sessionTTL = 24 * time.Hour`
  - `func NewStore(db *badger.DB) *Store`, `func HashPassword(string) ([]byte, error)`
  - `(*Store) PutUser(User) error`, `GetUser(string) (User, error)`, `ListUsers() ([]User, error)` (hashes stripped), `DeleteUser(string) error`, `Authenticate(user, pass string) (User, error)`
  - `(*Store) CreateSession(username string) (token string, err error)`, `SessionUser(token string) (string, error)`, `DeleteSession(token string) error`
  - `var usernameRe`, `func validPassword(string) bool`
  - Test helper `openMem(t) *badger.DB` (used by Tasks 4 and 5)

- [ ] **Step 1: Write the failing test**

```go
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
```

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./internal/gocron/api/auth/ -run TestStore -v`
Expected: FAIL, `undefined: NewStore`

- [ ] **Step 3: Implement `store.go`**

```go
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
```

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/gocron/api/auth/ -v`
Expected: PASS (`TestPolicy`, `TestStore`)

- [ ] **Step 5: Commit**

```bash
git add internal/gocron/api/auth/store.go internal/gocron/api/auth/store_test.go
git commit -m "feat(auth): store users and sessions in badger"
```

---

### Task 4: HTTP middleware and endpoints

**Files:**
- Modify: `pkg/core/response.go` (add two helpers)
- Create: `internal/gocron/api/auth/http.go`
- Test: `internal/gocron/api/auth/http_test.go`

**Interfaces:**
- Consumes: `NewEnforcer`, `Roles`, `Store` and its methods, `usernameRe`, `validPassword`, `sessionTTL`, `openMem` (Tasks 2–3)
- Produces:
  - `core.Unauthorized(c) error` (401), `core.Forbidden(c) error` (403)
  - `type Auth struct { Store *Store; Enforcer *casbin.Enforcer }`, `(*Auth) Routes(app *fiber.App)`
  - Endpoints: `POST /v1/auth/login` → `{"data":{"token","user"}}` plus cookie `gocron_session`; `POST /v1/auth/logout`; `GET /v1/auth/me` → `{"auth":true,"username","role","can":{"write_jobs","manage_users"}}`; `GET/POST /v1/users`; `PUT/DELETE /v1/users/:username`

- [ ] **Step 1: Add the response helpers** to `pkg/core/response.go` (same style as `NotFound`):

```go
func Unauthorized(c *fiber.Ctx) error {
	return c.Status(http.StatusUnauthorized).JSON(Response{
		Code:    fmt.Sprintf("%d", http.StatusUnauthorized),
		Message: http.StatusText(http.StatusUnauthorized),
	})
}

func Forbidden(c *fiber.Ctx) error {
	return c.Status(http.StatusForbidden).JSON(Response{
		Code:    fmt.Sprintf("%d", http.StatusForbidden),
		Message: http.StatusText(http.StatusForbidden),
	})
}
```

- [ ] **Step 2: Write the failing test**

`http_test.go` builds an app with the real middleware and stand-in `/v1/scheduler` handlers, and seeds the users directly. `Setup` doesn't exist until Task 5, so this test builds `Auth` by hand:

```go
package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func newTestApp(t *testing.T) *fiber.App {
	t.Helper()
	e, err := NewEnforcer()
	if err != nil {
		t.Fatal(err)
	}
	a := &Auth{Store: NewStore(openMem(t)), Enforcer: e}
	for name, role := range map[string]string{"admin": "admin", "vic": "viewer", "eddie": "editor"} {
		hash, _ := HashPassword("password1")
		if err := a.Store.PutUser(User{Username: name, Role: role, PasswordHash: hash}); err != nil {
			t.Fatal(err)
		}
	}
	app := fiber.New()
	a.Routes(app)
	ok := func(c *fiber.Ctx) error { return c.SendString("ok") }
	app.Get("/", ok) // stands in for the dashboard
	app.Get("/v1/scheduler", ok)
	app.Post("/v1/scheduler", ok)
	return app
}

func do(t *testing.T, app *fiber.App, method, path, token, body string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func loginAs(t *testing.T, app *fiber.App, user, pass string) string {
	t.Helper()
	res := do(t, app, "POST", "/v1/auth/login", "", fmt.Sprintf(`{"username":%q,"password":%q}`, user, pass))
	if res.StatusCode != 200 {
		t.Fatalf("login %s: status %d", user, res.StatusCode)
	}
	var body struct{ Data struct{ Token string } }
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil || body.Data.Token == "" {
		t.Fatalf("login %s: no token (%v)", user, err)
	}
	return body.Data.Token
}

func TestAuthFlow(t *testing.T) {
	app := newTestApp(t)
	expect := func(res *http.Response, want int, what string) {
		t.Helper()
		if res.StatusCode != want {
			t.Fatalf("%s: status %d, want %d", what, res.StatusCode, want)
		}
	}

	expect(do(t, app, "GET", "/v1/scheduler", "", ""), 401, "no token")
	expect(do(t, app, "GET", "/", "", ""), 200, "dashboard stays public")
	expect(do(t, app, "POST", "/v1/auth/login", "", `{"username":"vic","password":"wrong-pass"}`), 401, "wrong password")
	expect(do(t, app, "POST", "/v1/auth/login", "", `{"username":"ghost","password":"password1"}`), 401, "unknown user")

	viewer := loginAs(t, app, "vic", "password1")
	expect(do(t, app, "GET", "/v1/scheduler", viewer, ""), 200, "viewer reads jobs")
	expect(do(t, app, "POST", "/v1/scheduler", viewer, "{}"), 403, "viewer creates job")
	expect(do(t, app, "POST", "/V1/SCHEDULER/", viewer, "{}"), 403, "case/slash variants must not bypass casbin")
	expect(do(t, app, "GET", "/v1/users", viewer, ""), 403, "viewer lists users")

	editor := loginAs(t, app, "eddie", "password1")
	expect(do(t, app, "POST", "/v1/scheduler", editor, "{}"), 200, "editor creates job")

	admin := loginAs(t, app, "admin", "password1")
	expect(do(t, app, "POST", "/v1/users", admin, `{"username":"newbie","password":"password1","role":"viewer"}`), 200, "admin creates user")
	expect(do(t, app, "POST", "/v1/users", admin, `{"username":"newbie","password":"password1","role":"viewer"}`), 400, "duplicate user")
	expect(do(t, app, "POST", "/v1/users", admin, `{"username":"x","password":"short","role":"root"}`), 400, "invalid user")
	newbie := loginAs(t, app, "newbie", "password1")
	expect(do(t, app, "DELETE", "/v1/users/admin", admin, ""), 400, "admin deletes self")
	expect(do(t, app, "PUT", "/v1/users/admin", admin, `{"role":"viewer"}`), 400, "demote last admin")
	expect(do(t, app, "DELETE", "/v1/users/newbie", admin, ""), 200, "admin deletes user")
	expect(do(t, app, "GET", "/v1/scheduler", newbie, ""), 401, "deleted user's session")

	expect(do(t, app, "POST", "/v1/auth/logout", viewer, ""), 200, "logout")
	expect(do(t, app, "GET", "/v1/scheduler", viewer, ""), 401, "token after logout")
}

func TestMe(t *testing.T) {
	app := newTestApp(t)
	res := do(t, app, "GET", "/v1/auth/me", loginAs(t, app, "vic", "password1"), "")
	var body struct {
		Data struct {
			Auth     bool
			Username string
			Role     string
			Can      struct {
				WriteJobs   bool `json:"write_jobs"`
				ManageUsers bool `json:"manage_users"`
			}
		}
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	d := body.Data
	if !d.Auth || d.Username != "vic" || d.Role != "viewer" || d.Can.WriteJobs || d.Can.ManageUsers {
		t.Fatalf("me: %+v", d)
	}
}

func TestLoginCookie(t *testing.T) {
	res := do(t, newTestApp(t), "POST", "/v1/auth/login", "", `{"username":"vic","password":"password1"}`)
	cookie := res.Header.Get("Set-Cookie")
	for _, want := range []string{"gocron_session=", "HttpOnly", "SameSite=Strict"} {
		if !strings.Contains(cookie, want) {
			t.Errorf("Set-Cookie %q lacks %q", cookie, want)
		}
	}
}
```

- [ ] **Step 3: Run it and watch it fail**

Run: `go test ./internal/gocron/api/auth/ -run 'TestAuthFlow|TestMe|TestLoginCookie' -v`
Expected: FAIL, `undefined: Auth`

- [ ] **Step 4: Implement `http.go`**

```go
package auth

import (
	"slices"
	"strings"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/prongbang/gocron/pkg/core"
)

const cookieName = "gocron_session"

type Auth struct {
	Store    *Store
	Enforcer *casbin.Enforcer
}

// Routes installs the auth middleware and endpoints. Call it before registering the routes it protects.
func (a *Auth) Routes(app *fiber.App) {
	app.Use(a.middleware)
	v1 := app.Group("/v1")
	v1.Post("/auth/login", limiter.New(limiter.Config{Max: 10, Expiration: time.Minute}), a.login)
	v1.Post("/auth/logout", a.logout)
	v1.Get("/auth/me", a.me)
	v1.Get("/users", a.listUsers)
	v1.Post("/users", a.createUser)
	v1.Put("/users/:username", a.updateUser)
	v1.Delete("/users/:username", a.deleteUser)
}

func tokenFrom(c *fiber.Ctx) string {
	if h := c.Get(fiber.HeaderAuthorization); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return c.Cookies(cookieName)
}

// middleware authenticates every /v1 call except login, then asks casbin whether the caller's role may make it.
// fiber routes case-insensitively and ignores a trailing slash, so the path is normalized the same way before
// the check; otherwise "/V1/SCHEDULER/" would reach the handler without matching any policy line's intent.
func (a *Auth) middleware(c *fiber.Ctx) error {
	path := strings.TrimSuffix(strings.ToLower(c.Path()), "/")
	if path != "/v1" && !strings.HasPrefix(path, "/v1/") {
		return c.Next() // dashboard files are public; the data behind them is not
	}
	if path == "/v1/auth/login" {
		return c.Next()
	}
	name, err := a.Store.SessionUser(tokenFrom(c))
	if err != nil {
		return core.Unauthorized(c)
	}
	u, err := a.Store.GetUser(name) // re-read every time so role changes and deletions apply at once
	if err != nil {
		return core.Unauthorized(c)
	}
	c.Locals("user", u)
	if strings.HasPrefix(path, "/v1/auth/") {
		return c.Next() // me and logout: any signed-in user
	}
	if ok, err := a.Enforcer.Enforce(u.Role, path, c.Method()); err != nil || !ok {
		return core.Forbidden(c)
	}
	return c.Next()
}

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func public(u User) User {
	u.PasswordHash = nil
	return u
}

func (a *Auth) login(c *fiber.Ctx) error {
	var r credentials
	if err := c.BodyParser(&r); err != nil {
		return core.BadRequest(c, "invalid body")
	}
	u, err := a.Store.Authenticate(strings.TrimSpace(r.Username), r.Password)
	if err != nil {
		return core.Unauthorized(c)
	}
	token, err := a.Store.CreateSession(u.Username)
	if err != nil {
		return err
	}
	c.Cookie(&fiber.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(sessionTTL.Seconds()),
		HTTPOnly: true,
		SameSite: fiber.CookieSameSiteStrictMode,
		Secure:   c.Protocol() == "https",
	})
	return core.Ok(c, fiber.Map{"token": token, "user": public(u)})
}

func (a *Auth) logout(c *fiber.Ctx) error {
	_ = a.Store.DeleteSession(tokenFrom(c))
	c.ClearCookie(cookieName)
	return core.Ok(c, nil)
}

func (a *Auth) me(c *fiber.Ctx) error {
	u := c.Locals("user").(User)
	can := func(path, method string) bool {
		ok, _ := a.Enforcer.Enforce(u.Role, path, method)
		return ok
	}
	return core.Ok(c, fiber.Map{
		"auth":     true,
		"username": u.Username,
		"role":     u.Role,
		"can": fiber.Map{
			"write_jobs":   can("/v1/scheduler", "POST"),
			"manage_users": can("/v1/users", "GET"),
		},
	})
}

func (a *Auth) listUsers(c *fiber.Ctx) error {
	users, err := a.Store.ListUsers()
	if err != nil {
		return err
	}
	return core.Ok(c, users)
}

const (
	badUsername = "username must be 3-32 characters of a-z 0-9 _ . -"
	badPassword = "password must be 8-72 characters"
	badRole     = "role must be admin, editor or viewer"
)

func (a *Auth) createUser(c *fiber.Ctx) error {
	var r credentials
	if err := c.BodyParser(&r); err != nil {
		return core.BadRequest(c, "invalid body")
	}
	switch {
	case !usernameRe.MatchString(r.Username):
		return core.BadRequest(c, badUsername)
	case !validPassword(r.Password):
		return core.BadRequest(c, badPassword)
	case !slices.Contains(Roles, r.Role):
		return core.BadRequest(c, badRole)
	}
	if _, err := a.Store.GetUser(r.Username); err == nil {
		return core.BadRequest(c, "user already exists")
	}
	hash, err := HashPassword(r.Password)
	if err != nil {
		return err
	}
	u := User{Username: r.Username, Role: r.Role, PasswordHash: hash, CreatedAt: time.Now()}
	if err := a.Store.PutUser(u); err != nil {
		return err
	}
	return core.Created(c, public(u))
}

// updateUser changes role and/or password; empty fields are left as they are.
func (a *Auth) updateUser(c *fiber.Ctx) error {
	u, err := a.Store.GetUser(c.Params("username"))
	if err != nil {
		return core.NotFound(c, "user not found")
	}
	var r credentials
	if err := c.BodyParser(&r); err != nil {
		return core.BadRequest(c, "invalid body")
	}
	if r.Role != "" {
		if !slices.Contains(Roles, r.Role) {
			return core.BadRequest(c, badRole)
		}
		if u.Role == "admin" && r.Role != "admin" && a.adminCount() == 1 {
			return core.BadRequest(c, "cannot demote the last admin")
		}
		u.Role = r.Role
	}
	if r.Password != "" {
		if !validPassword(r.Password) {
			return core.BadRequest(c, badPassword)
		}
		if u.PasswordHash, err = HashPassword(r.Password); err != nil {
			return err
		}
	}
	if err := a.Store.PutUser(u); err != nil {
		return err
	}
	return core.Ok(c, public(u))
}

func (a *Auth) deleteUser(c *fiber.Ctx) error {
	name := c.Params("username")
	if name == c.Locals("user").(User).Username {
		return core.BadRequest(c, "you cannot delete yourself")
	}
	if _, err := a.Store.GetUser(name); err != nil {
		return core.NotFound(c, "user not found")
	}
	if err := a.Store.DeleteUser(name); err != nil {
		return err
	}
	return core.Ok(c, nil)
}

func (a *Auth) adminCount() int {
	users, _ := a.Store.ListUsers()
	n := 0
	for _, u := range users {
		if u.Role == "admin" {
			n++
		}
	}
	return n
}
```

Note: `core.Created` answers with HTTP 200 (existing behavior), which is why the test expects 200 for user creation.

- [ ] **Step 5: Run the tests**

Run: `go test ./internal/gocron/api/auth/ -v && go vet ./...`
Expected: PASS, no vet output

- [ ] **Step 6: Commit**

```bash
git add pkg/core/response.go internal/gocron/api/auth/http.go internal/gocron/api/auth/http_test.go
git commit -m "feat(auth): add login, sessions middleware and user management endpoints"
```

---

### Task 5: Setup, wiring and docs

**Files:**
- Create: `internal/gocron/api/auth/setup.go`
- Test: `internal/gocron/api/auth/setup_test.go`
- Modify: `internal/gocron/api/provider.go`, `internal/gocron/api/api.go`, `README.md`

**Interfaces:**
- Consumes: everything from Tasks 2–4
- Produces: `func Setup(db *badger.DB, adminUser, adminPassword string) (*Auth, error)`; `func NewAPI(router Routers, authn *auth.Auth) API`; env vars `GOCRON_AUTH`, `GOCRON_ADMIN_USER`, `GOCRON_ADMIN_PASSWORD`; with auth off, `GET /v1/auth/me` → `{"data":{"auth":false}}`

- [ ] **Step 1: Write the failing test**

```go
package auth

import "testing"

func TestSetupSeedsFirstAdminOnce(t *testing.T) {
	db := openMem(t)
	if _, err := Setup(db, "", ""); err == nil {
		t.Fatal("empty DB without admin credentials must refuse to start")
	}
	if _, err := Setup(db, "root", "first-pass"); err != nil {
		t.Fatal(err)
	}
	a, err := Setup(db, "root", "second-pass") // restart with changed env
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Store.Authenticate("root", "first-pass"); err != nil {
		t.Fatal("a restart must not reset the admin password")
	}
}
```

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./internal/gocron/api/auth/ -run TestSetup -v`
Expected: FAIL, `undefined: Setup`

- [ ] **Step 3: Implement `setup.go`**

```go
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
```

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/gocron/api/auth/ -v`
Expected: PASS

- [ ] **Step 5: Wire it in.** In `provider.go`, change the end of `CreateAPI` from

```go
	apiAPI := NewAPI(apiRouters)
	return apiAPI
```

to

```go
	return NewAPI(apiRouters, createAuth(dbDriver))
}

// createAuth returns nil when GOCRON_AUTH is off, which keeps the API open as before.
func createAuth(dbDriver database.Drivers) *auth.Auth {
	if on, _ := strconv.ParseBool(os.Getenv("GOCRON_AUTH")); !on {
		fmt.Println("[WARN] GOCRON_AUTH is off: anyone who can reach port 8000 can manage jobs")
		return nil
	}
	a, err := auth.Setup(dbDriver.BadgerDB(), os.Getenv("GOCRON_ADMIN_USER"), os.Getenv("GOCRON_ADMIN_PASSWORD"))
	if err != nil {
		log.Fatal("[ERROR] ", err)
	}
	return a
```

(add imports `fmt`, `log`, `os`, `strconv`, `github.com/prongbang/gocron/internal/gocron/api/auth`).

In `api.go`: add `Auth *auth.Auth` to `type api struct`, change `NewAPI` to

```go
func NewAPI(router Routers, authn *auth.Auth) API {
	return &api{
		Router: router,
		Auth:   authn,
	}
}
```

and in `Register`, directly above `// Routers`, insert:

```go
	// Auth must be registered before the routes it protects.
	if a.Auth != nil {
		a.Auth.Routes(app)
	} else {
		app.Get("/v1/auth/me", func(c *fiber.Ctx) error { return core.Ok(c, fiber.Map{"auth": false}) })
	}
```

(add imports `github.com/prongbang/gocron/internal/gocron/api/auth` and `github.com/prongbang/gocron/pkg/core`).

- [ ] **Step 6: Smoke-test the binary.** Run it from a scratch directory so its `./tmp/badger` stays out of the repo:

```bash
go build -o /tmp/gocron-auth ./cmd/gocron && mkdir -p /tmp/gocron-auth-run && cd /tmp/gocron-auth-run
GOCRON_API=true GOCRON_AUTH=true GOCRON_ADMIN_USER=admin GOCRON_ADMIN_PASSWORD=change-me-now /tmp/gocron-auth &
sleep 2
curl -s -o /dev/null -w '%{http_code}\n' localhost:8000/v1/scheduler                    # 401
TOKEN=$(curl -s localhost:8000/v1/auth/login -H 'Content-Type: application/json' -d '{"username":"admin","password":"change-me-now"}' | python3 -c 'import json,sys;print(json.load(sys.stdin)["data"]["token"])')
curl -s -o /dev/null -w '%{http_code}\n' -H "Authorization: Bearer $TOKEN" localhost:8000/v1/scheduler   # 200
kill %1
```

Expected: `401`, then `200`. Then start without `GOCRON_AUTH` and check that `curl localhost:8000/v1/auth/me` prints `{"code":"200","message":"OK","data":{"auth":false}}`.

- [ ] **Step 7: Document it.** In `README.md`, after the `### Web dashboard` section, add:

````markdown
### Authentication

Off by default. Turn it on with:

```yaml
    environment:
      - GOCRON_API=true
      - GOCRON_AUTH=true
      - GOCRON_ADMIN_USER=admin          # first admin, created only when no user exists
      - GOCRON_ADMIN_PASSWORD=change-me  # 8-72 characters
```

| Role | Can |
|---|---|
| `viewer` | See jobs and history |
| `editor` | Also create and stop jobs |
| `admin` | Also manage users (`/v1/users`) |

The dashboard signs in with a cookie. Scripts get a token (valid 24h) and send it as a Bearer header:

```shell
TOKEN=$(curl -s localhost:8000/v1/auth/login -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"change-me"}' | jq -r .data.token)
curl -H "Authorization: Bearer $TOKEN" localhost:8000/v1/scheduler
```

Put gocron behind HTTPS (a reverse proxy) when it is reachable from outside your machine.
````

- [ ] **Step 8: Run everything and commit**

```bash
go test ./... && go vet ./...
git add internal/gocron/api/auth/setup.go internal/gocron/api/auth/setup_test.go internal/gocron/api/provider.go internal/gocron/api/api.go README.md
git commit -m "feat(api): protect the api with GOCRON_AUTH and bootstrap the first admin"
```

---

### Task 6: Dashboard sign-in

**Files:**
- Modify: `web/src/lib/api.ts`, `web/src/routes/+layout.svelte`, `web/src/routes/+page.svelte`, `web/src/lib/components/job-table.svelte`
- Create: `web/src/lib/session.svelte.ts`, `web/src/routes/login/+page.svelte`

**Interfaces:**
- Consumes: endpoints from Tasks 4–5
- Produces: `type Me`, `getMe()`, `login()`, `logout()` in `api.ts`; `session`, `refreshSession()`, `canWriteJobs()` in `session.svelte.ts`

- [ ] **Step 1: `api.ts`.** In `call`, right after `const text = await res.text();`, add:

```ts
	// Signed out or session expired: send the user to the login page and come back afterwards.
	if (res.status === 401 && !path.startsWith('/v1/auth/')) {
		location.href = `/login?next=${encodeURIComponent(location.pathname + location.search)}`;
		throw new Error('Signed out');
	}
```

and append at the end of the file:

```ts
export type Me =
	| { auth: false }
	| {
			auth: true;
			username: string;
			role: string;
			can: { write_jobs: boolean; manage_users: boolean };
	  };

export const getMe = () => call<Me>('/v1/auth/me');

export const login = (username: string, password: string) =>
	call<{ token: string }>('/v1/auth/login', {
		method: 'POST',
		body: JSON.stringify({ username, password })
	});

export const logout = () => call<null>('/v1/auth/logout', { method: 'POST' });
```

- [ ] **Step 2: Create `session.svelte.ts`**

```ts
import { getMe, type Me } from '$lib/api';

// Who is signed in; null until /v1/auth/me answers.
export const session = $state<{ me: Me | null }>({ me: null });

export async function refreshSession() {
	session.me = await getMe();
}

/** True when auth is off, or the signed-in role may create and stop jobs. */
export const canWriteJobs = () =>
	session.me?.auth === false || (session.me?.auth === true && session.me.can.write_jobs);
```

- [ ] **Step 3: Create `routes/login/+page.svelte`**

```svelte
<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import * as Card from '$lib/components/ui/card';
	import * as Field from '$lib/components/ui/field';
	import * as Alert from '$lib/components/ui/alert';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';
	import { Spinner } from '$lib/components/ui/spinner';
	import TriangleAlertIcon from '@lucide/svelte/icons/triangle-alert';
	import logo from '$lib/assets/logo.svg';
	import { login } from '$lib/api';

	let username = $state('');
	let password = $state('');
	let error = $state('');
	let busy = $state(false);

	// Same-site paths only, so ?next= cannot bounce people to another site.
	const next = $derived.by(() => {
		const n = page.url.searchParams.get('next') ?? '/';
		return n.startsWith('/') && !n.startsWith('//') ? n : '/';
	});

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = '';
		try {
			await login(username.trim(), password);
			await goto(next);
		} catch (err) {
			const msg = (err as Error).message;
			error = msg === 'Unauthorized' ? 'Wrong username or password.' : msg;
		} finally {
			busy = false;
		}
	}
</script>

<main class="flex min-h-svh items-center justify-center p-4">
	<Card.Root class="w-full max-w-sm">
		<Card.Header class="text-center">
			<img src={logo} alt="" class="mx-auto size-10" />
			<Card.Title>Sign in to gocron</Card.Title>
		</Card.Header>
		<Card.Content>
			<form onsubmit={submit} class="flex flex-col gap-6">
				{#if error}
					<Alert.Root variant="destructive">
						<TriangleAlertIcon />
						<Alert.Title>{error}</Alert.Title>
					</Alert.Root>
				{/if}
				<Field.Group>
					<Field.Field>
						<Field.Label for="username">Username</Field.Label>
						<Input id="username" autocomplete="username" required bind:value={username} />
					</Field.Field>
					<Field.Field>
						<Field.Label for="password">Password</Field.Label>
						<Input
							id="password"
							type="password"
							autocomplete="current-password"
							required
							bind:value={password}
						/>
					</Field.Field>
				</Field.Group>
				<Button type="submit" disabled={busy}>
					{#if busy}<Spinner data-icon="inline-start" />{/if}
					Sign in
				</Button>
			</form>
		</Card.Content>
	</Card.Root>
</main>
```

- [ ] **Step 4: Layout.** In `+layout.svelte`:

Add these imports:

```ts
	import { goto } from '$app/navigation';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import LogOutIcon from '@lucide/svelte/icons/log-out';
	import { logout } from '$lib/api';
	import { session, refreshSession } from '$lib/session.svelte';
```

Replace `const links = [...]` with:

```ts
	const onLogin = $derived(page.url.pathname === '/login');
	$effect(() => {
		if (!onLogin) refreshSession().catch(() => {});
	});

	const links = $derived([
		{ href: '/', label: 'Jobs' },
		{ href: '/history', label: 'History' },
		...(session.me?.auth && session.me.can.manage_users ? [{ href: '/users', label: 'Users' }] : [])
	]);

	async function signOut() {
		await logout().catch(() => {});
		session.me = null;
		goto('/login');
	}
```

Then wrap the whole `<nav>…</nav>` in `{#if !onLogin} … {/if}`. Inside the nav's inner `div`, after the `<div class="flex gap-1">…</div>` that holds the links, add:

```svelte
			{#if session.me?.auth}
				<div class="ml-auto flex items-center gap-2">
					<span class="text-sm">{session.me.username}</span>
					<Badge variant="secondary">{session.me.role}</Badge>
					<Button variant="ghost" size="sm" onclick={signOut}>
						<LogOutIcon data-icon="inline-start" />
						Sign out
					</Button>
				</div>
			{/if}
```

- [ ] **Step 5: Hide write actions from viewers.** In `routes/+page.svelte`, add `import { canWriteJobs } from '$lib/session.svelte';` and change `<CreateJobDialog {projects} oncreated={load} />` to:

```svelte
			{#if canWriteJobs()}<CreateJobDialog {projects} oncreated={load} />{/if}
```

In `job-table.svelte`, add the same import and wrap the whole `<AlertDialog.Root>…</AlertDialog.Root>` (the Stop button) in `{#if canWriteJobs()} … {/if}`. The server rejects the calls anyway; this only hides buttons that would fail.

- [ ] **Step 6: Check and verify in the browser**

Run: `cd web && bun run check`. Expected: `0 ERRORS 0 WARNINGS`.

Then start gocron with auth on (Task 5, Step 6) and `bun run dev` (it proxies `/v1` to :8000). Check each of these:
1. `http://localhost:5173/history` redirects to `/login?next=%2Fhistory`, with no nav bar.
2. A wrong password shows "Wrong username or password.".
3. `admin` / `change-me-now` lands on `/history`, and the nav shows `admin`, an `admin` badge and Sign out.
4. Sign out returns to `/login`, and `/` then redirects back to the login page.

- [ ] **Step 7: Commit**

```bash
git add web/src/lib/api.ts web/src/lib/session.svelte.ts web/src/routes/login/+page.svelte web/src/routes/+layout.svelte web/src/routes/+page.svelte web/src/lib/components/job-table.svelte
git commit -m "feat(web): add sign-in page and hide job actions from viewers"
```

---

### Task 7: Users page for admins

**Files:**
- Modify: `web/src/lib/api.ts`
- Create: `web/src/lib/components/create-user-dialog.svelte`, `web/src/routes/users/+page.svelte`

**Interfaces:**
- Consumes: `/v1/users` endpoints (Task 4), `session` (Task 6)
- Produces: `ROLES`, `type User`, `listUsers`, `createUser`, `updateUser`, `deleteUser` in `api.ts`

- [ ] **Step 1: Client calls.** Append to `api.ts`:

```ts
export const ROLES = ['admin', 'editor', 'viewer'] as const;

export type User = { username: string; role: string; created_at: string };

export const listUsers = async () => (await call<User[] | null>('/v1/users')) ?? [];

export const createUser = (u: { username: string; password: string; role: string }) =>
	call<User>('/v1/users', { method: 'POST', body: JSON.stringify(u) });

export const updateUser = (username: string, patch: { role?: string; password?: string }) =>
	call<User>(`/v1/users/${encodeURIComponent(username)}`, {
		method: 'PUT',
		body: JSON.stringify(patch)
	});

export const deleteUser = (username: string) =>
	call<null>(`/v1/users/${encodeURIComponent(username)}`, { method: 'DELETE' });
```

- [ ] **Step 2: Create `create-user-dialog.svelte`**

```svelte
<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog';
	import * as Field from '$lib/components/ui/field';
	import * as Select from '$lib/components/ui/select';
	import { Button, buttonVariants } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Spinner } from '$lib/components/ui/spinner';
	import UserPlusIcon from '@lucide/svelte/icons/user-plus';
	import { toast } from 'svelte-sonner';
	import { ROLES, createUser } from '$lib/api';

	let { oncreated }: { oncreated: () => void } = $props();

	let open = $state(false);
	let saving = $state(false);
	let username = $state('');
	let password = $state('');
	let role = $state('viewer');

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		saving = true;
		try {
			await createUser({ username: username.trim(), password, role });
			toast.success('User created', { description: username });
			open = false;
			username = password = '';
			oncreated();
		} catch (err) {
			toast.error('Create failed', { description: (err as Error).message });
		} finally {
			saving = false;
		}
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Trigger class={buttonVariants()}>
		<UserPlusIcon data-icon="inline-start" />
		New user
	</Dialog.Trigger>
	<Dialog.Content class="sm:max-w-md">
		<form onsubmit={submit} class="flex flex-col gap-6">
			<Dialog.Header>
				<Dialog.Title>New user</Dialog.Title>
				<Dialog.Description>
					Viewers can look, editors can also create and stop jobs, admins can also manage users.
				</Dialog.Description>
			</Dialog.Header>
			<Field.Group>
				<Field.Field>
					<Field.Label for="new-username">Username</Field.Label>
					<Input id="new-username" required pattern={'[a-z0-9_.\\-]{3,32}'} bind:value={username} />
					<Field.Description>3–32 characters: a–z, 0–9, _ . -</Field.Description>
				</Field.Field>
				<Field.Field>
					<Field.Label for="new-password">Password</Field.Label>
					<Input
						id="new-password"
						type="password"
						autocomplete="new-password"
						required
						minlength={8}
						maxlength={72}
						bind:value={password}
					/>
				</Field.Field>
				<Field.Field>
					<Field.Label for="new-role">Role</Field.Label>
					<Select.Root type="single" bind:value={role}>
						<Select.Trigger id="new-role" class="w-full">{role}</Select.Trigger>
						<Select.Content>
							<Select.Group>
								{#each ROLES as r (r)}<Select.Item value={r}>{r}</Select.Item>{/each}
							</Select.Group>
						</Select.Content>
					</Select.Root>
				</Field.Field>
			</Field.Group>
			<Dialog.Footer>
				<Dialog.Close class={buttonVariants({ variant: 'outline' })}>Cancel</Dialog.Close>
				<Button type="submit" disabled={saving}>
					{#if saving}<Spinner data-icon="inline-start" />{/if}
					Create
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
```

- [ ] **Step 3: Create `routes/users/+page.svelte`**

```svelte
<script lang="ts">
	import { onMount } from 'svelte';
	import * as Card from '$lib/components/ui/card';
	import * as Table from '$lib/components/ui/table';
	import * as Select from '$lib/components/ui/select';
	import * as AlertDialog from '$lib/components/ui/alert-dialog';
	import * as Alert from '$lib/components/ui/alert';
	import { Badge } from '$lib/components/ui/badge';
	import { buttonVariants } from '$lib/components/ui/button';
	import Trash2Icon from '@lucide/svelte/icons/trash-2';
	import TriangleAlertIcon from '@lucide/svelte/icons/triangle-alert';
	import { toast } from 'svelte-sonner';
	import CreateUserDialog from '$lib/components/create-user-dialog.svelte';
	import { ROLES, deleteUser, listUsers, updateUser, type User } from '$lib/api';
	import { session } from '$lib/session.svelte';

	let users = $state<User[]>([]);
	let error = $state('');
	const me = $derived(session.me?.auth ? session.me.username : '');

	async function load() {
		try {
			users = (await listUsers()).sort((a, b) => a.username.localeCompare(b.username));
			error = '';
		} catch (err) {
			error = (err as Error).message;
		}
	}

	async function run(action: () => Promise<unknown>, done: string) {
		try {
			await action();
			toast.success(done);
		} catch (err) {
			toast.error((err as Error).message);
		}
		await load();
	}

	onMount(load);
</script>

<main class="mx-auto flex max-w-6xl flex-col gap-6 p-4 sm:p-8">
	<header class="flex flex-wrap items-center justify-between gap-4">
		<div class="flex flex-col gap-1">
			<h1 class="text-2xl font-semibold tracking-tight">Users</h1>
			<p class="text-muted-foreground text-sm">{users.length} users</p>
		</div>
		<CreateUserDialog oncreated={load} />
	</header>

	{#if error}
		<Alert.Root variant="destructive">
			<TriangleAlertIcon />
			<Alert.Title>Cannot load users</Alert.Title>
			<Alert.Description>{error}</Alert.Description>
		</Alert.Root>
	{/if}

	<Card.Root>
		<Card.Content>
			<Table.Root>
				<Table.Header>
					<Table.Row>
						<Table.Head>Username</Table.Head>
						<Table.Head>Role</Table.Head>
						<Table.Head>Created</Table.Head>
						<Table.Head class="text-right">Action</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each users as u (u.username)}
						<Table.Row>
							<Table.Cell>
								<div class="flex items-center gap-2">
									{u.username}
									{#if u.username === me}<Badge variant="outline">you</Badge>{/if}
								</div>
							</Table.Cell>
							<Table.Cell>
								<Select.Root
									type="single"
									value={u.role}
									disabled={u.username === me}
									onValueChange={(role) =>
										run(() => updateUser(u.username, { role }), `${u.username} is now ${role}`)}
								>
									<Select.Trigger class="w-32" aria-label="Role of {u.username}">{u.role}</Select.Trigger>
									<Select.Content>
										<Select.Group>
											{#each ROLES as r (r)}<Select.Item value={r}>{r}</Select.Item>{/each}
										</Select.Group>
									</Select.Content>
								</Select.Root>
							</Table.Cell>
							<Table.Cell class="text-muted-foreground">
								{new Date(u.created_at).toLocaleDateString(undefined, { dateStyle: 'medium' })}
							</Table.Cell>
							<Table.Cell class="text-right">
								{#if u.username !== me}
									<AlertDialog.Root>
										<AlertDialog.Trigger class={buttonVariants({ variant: 'ghost', size: 'sm' })}>
											<Trash2Icon data-icon="inline-start" />
											Delete
										</AlertDialog.Trigger>
										<AlertDialog.Content>
											<AlertDialog.Header>
												<AlertDialog.Title>Delete {u.username}?</AlertDialog.Title>
												<AlertDialog.Description>
													They are signed out and can no longer sign in. Jobs they created keep running.
												</AlertDialog.Description>
											</AlertDialog.Header>
											<AlertDialog.Footer>
												<AlertDialog.Cancel>Cancel</AlertDialog.Cancel>
												<AlertDialog.Action
													variant="destructive"
													onclick={() => run(() => deleteUser(u.username), `${u.username} deleted`)}
												>
													Delete user
												</AlertDialog.Action>
											</AlertDialog.Footer>
										</AlertDialog.Content>
									</AlertDialog.Root>
								{/if}
							</Table.Cell>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
		</Card.Content>
	</Card.Root>
</main>
```

- [ ] **Step 4: Check and verify in the browser**

Run: `cd web && bun run check`. Expected: `0 ERRORS 0 WARNINGS`.

Signed in as `admin`:
1. The nav shows **Users**.
2. Create `vic` as viewer; they appear in the table.
3. Your own row shows the "you" badge, a disabled role select and no Delete button.

Sign in as `vic` in a private window:
4. There's no Users link, no New job button and no Stop button.
5. Opening `/users` directly shows "Cannot load users" with `Forbidden`.

Back as admin:
6. Change `vic` to editor. Their New job button appears after a reload.
7. Delete `vic`. Their next request redirects them to `/login`.

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/api.ts web/src/lib/components/create-user-dialog.svelte web/src/routes/users/+page.svelte
git commit -m "feat(web): add users page for admins"
```

---

### Task 8: End-to-end check with the embedded build

**Files:** none (verification only)

- [ ] **Step 1: Build the real binary** with `make build`. Expected: `bin/gocron` exists.
- [ ] **Step 2: Run the Task 5 Step 6 commands against `bin/gocron`.** Also open `http://localhost:8000/` in a browser. It should show the login page served by Go, and signing in should land on Jobs.
- [ ] **Step 3: Check that auth-off still behaves like v1.1.x.** Start without `GOCRON_AUTH`. The dashboard opens with no login, and there's no user badge or Users link.
- [ ] **Step 4: Run the full suite.** Run `go test ./... && go vet ./... && (cd web && bun run check)`. Expected: all pass.

---

## Not in this plan (follow-ups)

- **Per-project roles** ("alice may edit only `billing`"). Use Casbin's RBAC-with-domains model, with `g = _, _, _` and the matcher `g(r.sub, p.sub, r.dom) && …`. The request's domain is the job's `project`, so the middleware has to load the job, or the handlers have to check it. History and job lists would also need filtering to the caller's projects. This is a separate plan.
- **Long-lived API tokens for CI**, so scripts don't have to log in every 24h. Add `token:` records created from the Users page.
- **SSO/OIDC, audit log, and a self-service password change screen.** `PUT /v1/users/:name` already accepts `password`, so an admin can reset one.
- **Editable policy without a rebuild.** Load `policy.csv` from a path in `GOCRON_POLICY`, falling back to the embedded copy.
