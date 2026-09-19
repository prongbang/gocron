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
