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
