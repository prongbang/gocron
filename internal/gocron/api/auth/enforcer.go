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
