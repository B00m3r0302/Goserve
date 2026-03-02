package auth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/alexedwards/argon2id"
)

func HashPassword(password string) (string, error) {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "Unable to hash", err
	}
	return string(hash), err
}

func CheckPasswordHash(password, hash string) (bool, error) {
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false, err
	}
	return match, nil
}

func GetAPIKey(headers http.Header) (string, error) {
	result := headers.Get("Authorization")
	match := strings.HasPrefix(result, "ApiKey ")
	if !match {
		return "", fmt.Errorf("invalid authorization header")
	}
	ApiKey := strings.TrimPrefix(result, "ApiKey ")
	ApiKey = strings.TrimSpace(ApiKey)
	return ApiKey, nil
}
