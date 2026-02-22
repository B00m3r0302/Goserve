package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
)

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) createUser(w http.ResponseWriter, r *http.Request) {
	type data struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	dat := data{}
	err := decoder.Decode(&dat)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		message := map[string]string{"error": "Something went wrong"}
		errDat, err := json.Marshal(message)
		if err != nil {
			panic(err)
		}
		w.Write(errDat)
		return
	}

	newUser, err := cfg.dbQueries.CreateUser(r.Context(), dat.Email)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		message := map[string]string{"error": "Something went wrong"}
		errDat, err := json.Marshal(message)
		if err != nil {
			panic(err)
		}
		w.Write(errDat)
		return
	}
	response := User{
		ID:        newUser.ID,
		CreatedAt: newUser.CreatedAt,
		UpdatedAt: newUser.UpdatedAt,
		Email:     newUser.Email,
	}

	returnMsg, err := json.Marshal(response)
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	w.Write(returnMsg)

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

func (cfg *apiConfig) createChirp(w http.ResponseWriter, r *http.Request) {
	type input struct {
		Body   string `json:"body"`
		UserId string `json:"user_id"`
	}

	type broken struct {
		Error string `json:"error"`
	}

	type valid struct {
		Valid bool `json:"valid"`
	}

	decoder := json.NewDecoder(r.Body)
	dat := input{}
	err := decoder.Decode(&dat)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		message := broken{
			Error: "Something went wrong",
		}
		errDat, err := json.Marshal(message)
		if err != nil {
			panic(err)
		}
		w.Write(errDat)
		return
	}

	if len(dat.Body) > 140 {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		message := broken{
			Error: "Chirp is too long",
		}
		badDat, err := json.Marshal(message)
		if err != nil {
			panic(err)
		}
		w.Write(badDat)
		return
	}

	type cleanedBody struct {
		CleanedBody string `json:"cleaned_body"`
	}
	// Find and replace strings
	ker := regexp.MustCompile(`(?i)kerfuffle`)
	keresult := ker.ReplaceAllString(dat.Body, "****")

	shar := regexp.MustCompile(`(?i)sharbert`)
	sharesult := shar.ReplaceAllString(keresult, "****")

	forn := regexp.MustCompile(`(?i)fornax`)
	fornresult := forn.ReplaceAllString(sharesult, "****")

	cleanedResult := cleanedBody{CleanedBody: fornresult}
	cleanedDat, err := json.Marshal(cleanedResult)
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(cleanedDat))
}
