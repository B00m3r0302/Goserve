package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("mysecretpassword")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword returned empty hash")
	}
	if hash == "mysecretpassword" {
		t.Fatal("HashPassword returned plaintext password")
	}
}

func TestCheckPasswordHash(t *testing.T) {
	password := "mysecretpassword"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	t.Run("correct password matches", func(t *testing.T) {
		match, err := CheckPasswordHash(password, hash)
		if err != nil {
			t.Fatalf("CheckPasswordHash returned error: %v", err)
		}
		if !match {
			t.Fatal("expected correct password to match")
		}
	})

	t.Run("wrong password does not match", func(t *testing.T) {
		match, err := CheckPasswordHash("wrongpassword", hash)
		if err != nil {
			t.Fatalf("CheckPasswordHash returned error: %v", err)
		}
		if match {
			t.Fatal("expected wrong password to not match")
		}
	})
}

func TestMakeJWT(t *testing.T) {
	userID := uuid.New()
	secret := "testsecret"

	token, err := MakeJWT(userID, secret, time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}
	if token == "" {
		t.Fatal("MakeJWT returned empty token")
	}
}

func TestValidateJWT(t *testing.T) {
	userID := uuid.New()
	secret := "testsecret"

	t.Run("valid token returns correct user ID", func(t *testing.T) {
		token, err := MakeJWT(userID, secret, time.Hour)
		if err != nil {
			t.Fatalf("MakeJWT returned error: %v", err)
		}

		gotID, err := ValidateJWT(token, secret)
		if err != nil {
			t.Fatalf("ValidateJWT returned error: %v", err)
		}
		if gotID != userID {
			t.Fatalf("expected user ID %v, got %v", userID, gotID)
		}
	})

	t.Run("wrong secret is rejected", func(t *testing.T) {
		token, err := MakeJWT(userID, secret, time.Hour)
		if err != nil {
			t.Fatalf("MakeJWT returned error: %v", err)
		}

		_, err = ValidateJWT(token, "wrongsecret")
		if err == nil {
			t.Fatal("expected error for wrong secret, got nil")
		}
	})

	t.Run("expired token is rejected", func(t *testing.T) {
		token, err := MakeJWT(userID, secret, -time.Second)
		if err != nil {
			t.Fatalf("MakeJWT returned error: %v", err)
		}

		_, err = ValidateJWT(token, secret)
		if err == nil {
			t.Fatal("expected error for expired token, got nil")
		}
	})

	t.Run("malformed token is rejected", func(t *testing.T) {
		_, err := ValidateJWT("this.is.not.a.token", secret)
		if err == nil {
			t.Fatal("expected error for malformed token, got nil")
		}
	})
}

func TestGetBearerToken(t *testing.T) {
	t.Run("valid bearer token", func(t *testing.T) {
		headers := http.Header{}
		headers["Authorization"] = []string{"Bearer", "mytoken123"}
		token, err := GetBearerToken(headers)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if token != "mytoken123" {
			t.Fatalf("expected token 'mytoken123', got '%s'", token)
		}
	})

	t.Run("empty headers returns empty token", func(t *testing.T) {
		headers := http.Header{}
		token, err := GetBearerToken(headers)
		if err != nil {
			t.Fatalf("expected no error for empty headers, got: %v", err)
		}
		if token != "" {
			t.Fatalf("expected empty token, got '%s'", token)
		}
	})

	t.Run("non-bearer auth scheme returns error", func(t *testing.T) {
		headers := http.Header{}
		headers["Authorization"] = []string{"Basic", "credentials"}
		_, err := GetBearerToken(headers)
		if err == nil {
			t.Fatal("expected error for non-bearer auth, got nil")
		}
	})

	t.Run("standard single-value bearer header returns error", func(t *testing.T) {
		// The current implementation expects two separate values ["Bearer", "<token>"],
		// not the standard single-value format "Bearer <token>".
		headers := http.Header{}
		headers["Authorization"] = []string{"Bearer mytoken"}
		_, err := GetBearerToken(headers)
		if err == nil {
			t.Fatal("expected error for single-value auth header, got nil")
		}
	})
}