package security

import (
	"bytes"
	"crypto/rand"
	"errors"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    = 1
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32
)

var ErrInvalidPassword = errors.New("invalid password")

// HashPassword hashes a plaintext password with a random salt using Argon2id.
func HashPassword(password string) (hash []byte, salt []byte, err error) {
	salt, err = generateSalt()
	if err != nil {
		return nil, nil, err
	}
	hash = argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return hash, salt, nil
}

// VerifyPassword checks a plaintext password against stored hash and salt.
func VerifyPassword(password string, storedHash, storedSalt []byte) error {
	computed := argon2.IDKey([]byte(password), storedSalt, argonTime, argonMemory, argonThreads, argonKeyLen)
	if !bytes.Equal(computed, storedHash) {
		return ErrInvalidPassword
	}
	return nil
}

func generateSalt() ([]byte, error) {
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	return salt, err
}
