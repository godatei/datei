package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	recoveryCodeLength = 10
	recoveryCodeCount  = 10
)

// GenerateRecoveryCodes generates 10 random recovery codes.
func GenerateRecoveryCodes() ([]string, error) {
	codes := make([]string, recoveryCodeCount)
	for i := range recoveryCodeCount {
		code, err := generateRecoveryCode()
		if err != nil {
			return nil, err
		}
		codes[i] = code
	}
	return codes, nil
}

func generateRecoveryCode() (string, error) {
	codeBytes := make([]byte, 6)
	if _, err := rand.Read(codeBytes); err != nil {
		return "", err
	}
	hexString := hex.EncodeToString(codeBytes)
	return hexString[:recoveryCodeLength], nil
}

// FormatRecoveryCode formats "abcde12345" as "abcde-12345".
func FormatRecoveryCode(code string) string {
	if len(code) != recoveryCodeLength {
		return code
	}
	return code[:5] + "-" + code[5:]
}

// NormalizeRecoveryCode removes dashes and lowercases.
func NormalizeRecoveryCode(code string) string {
	return strings.ToLower(strings.ReplaceAll(code, "-", ""))
}

// HashRecoveryCode hashes a recovery code with a random salt.
func HashRecoveryCode(code string) (hash []byte, salt []byte, err error) {
	salt, err = generateSalt()
	if err != nil {
		return nil, nil, err
	}
	normalized := NormalizeRecoveryCode(code)
	hash = argon2.IDKey([]byte(normalized), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return hash, salt, nil
}

// VerifyRecoveryCode checks a code against stored hash and salt.
func VerifyRecoveryCode(code string, storedHash, storedSalt []byte) bool {
	normalized := NormalizeRecoveryCode(code)
	computed := argon2.IDKey([]byte(normalized), storedSalt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return subtle.ConstantTimeCompare(computed, storedHash) == 1
}
