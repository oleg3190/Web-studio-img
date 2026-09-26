package auth

import (
    "errors"
    "fmt"
    "golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12
var ErrInvalidCredentials = errors.New("invalid credentials")

func HashPassword(password string) (string, error) {
    if len(password) < 8 { return "", errors.New("password must be at least 8 characters") }
    if len(password) > 72 { return "", errors.New("password must be at most 72 bytes") }
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
    if err != nil { return "", fmt.Errorf("hash password: %w", err) }
    return string(hash), nil
}
func CheckPassword(hash, password string) error {
    if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil { return ErrInvalidCredentials }
    return nil
}
