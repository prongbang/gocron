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
