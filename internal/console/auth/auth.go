package auth

import (
	"log/slog"

	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"golang.org/x/crypto/bcrypt"
)

type Password []byte

// NewPassword creates a new password from a plain text string.
func NewPassword(text string) (Password, error) {
	// TODO: Use salt and pepper
	pwd, err := bcrypt.GenerateFromPassword([]byte(text), 14)
	if err != nil {
		slog.Warn("Could not hash password", logging.Error(err))
	}
	return pwd, err
}

// String returns the password as a string.
func (p Password) String() string {
	return string(p)
}

// CheckPassword checks if a password matches a hash.
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
