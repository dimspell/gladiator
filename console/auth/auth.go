package auth

import (
	"log/slog"

	"golang.org/x/crypto/bcrypt"
)

type Password []byte

func NewPassword(text string) (Password, error) {
	pwd, err := bcrypt.GenerateFromPassword([]byte(text), 14)
	if err != nil {
		slog.Warn("Could not hash password", "error", err)
	}
	return pwd, err
}

func (p Password) String() string {
	return string(p)
}
