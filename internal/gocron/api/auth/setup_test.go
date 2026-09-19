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
