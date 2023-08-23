package common

import (
	"strings"

	"github.com/google/uuid"
)

func Uuid() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}
