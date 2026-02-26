package main

import (
	"log"
	"time"

	"github.com/B00m3r0302/Goserve/internal/auth"
	_ "github.com/lib/pq"
)

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) hitsCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) hitsReset(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("PLATFORM") == "dev" {
		err := cfg.dbQueries.ResetUsers(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Something went wrong while trying to reset the users table...\n"))
			w.Write([]byte(err.Error()))
		}
		return
	}
	w.WriteHeader(http.StatusForbidden)
}

func (cfg *apiConfig) refreshServer(w http.ResponseWriter, r *http.Request) {
	type newToken struct {
		Token string `json:"token"`
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("{\"message\": \"No token provided\"}"))
		log.Printf("Error getting JWT, was not provided: %s", err)
		return
	}

	// Check if expired or revoked
	results, err := cfg.dbQueries.GetTokenExpiresRevokeByToken(r.Context(), token)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong while trying to get the token values...\n"))
		w.Write([]byte(err.Error()))
		log.Printf("Something went wrong while trying to get the token values: %s", err)
		return
	}

	if results.ExpiresAt.Before(time.Now().Add(1*time.Hour)) || results.RevokedAt.Valid {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("{\"message\": \"Token expired or revoked\"}"))
		log.Printf("Token expired or revoked: %s", err)
		return
	}

	newJWT, err := auth.MakeJWT(results.UserID, cfg.secretKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong while trying to generate the JWT...\n"))
		w.Write([]byte(err.Error()))
		log.Printf("Something went wrong while trying to generate the JWT: %s", err)
		return
	}

	// validate JWT
	_, err = auth.ValidateJWT(newJWT, cfg.secretKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong while trying to validate the JWT...\n"))
		w.Write([]byte(err.Error()))
		log.Printf("Something went wrong while trying to validate the JWT: %s", err)
		return
	}

	response := newToken{
		Token: newJWT,
	}

	final, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong while trying to encode the response...\n"))
		w.Write([]byte(err.Error()))
		log.Printf("Something went wrong while trying to encode the response: %s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(final)

}

func (cfg *apiConfig) revokeToken(w http.ResponseWriter, r *http.Request) {
	// Check for token in header
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("{\"message\": \"No token provided\"}"))
		log.Printf("Error getting JWT, was not provided: %s", err)
		return
	}

	err = cfg.dbQueries.RevokeToken(r.Context(), token)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong while trying to revoke the token...\n"))
		w.Write([]byte(err.Error()))
		log.Printf("Something went wrong while trying to revoke the token: %s", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
