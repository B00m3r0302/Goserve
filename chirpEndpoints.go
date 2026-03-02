package main

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"sort"
	"time"

	"github.com/B00m3r0302/Goserve/internal/auth"
	"github.com/B00m3r0302/Goserve/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) createChirp(w http.ResponseWriter, r *http.Request) {
	type input struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	type broken struct {
		Error string `json:"error"`
	}

	type chirpResponse struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserID    uuid.UUID `json:"user_id"`
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
		errDat, _ := json.Marshal(message)
		w.Write(errDat)
		return
	}

	jwt, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("{\"message\": \"Unauthorized\"}"))
		log.Printf("Error getting JWT: %s", err)
		return
	}

	valid, err := auth.ValidateJWT(jwt, cfg.secretKey)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("{\"message\": \"Unauthorized\"}"))
		log.Printf("Error validating JWT: %s", err)
		return
	}

	if len(dat.Body) > 140 {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		message := broken{
			Error: "Chirp is too long",
		}
		badDat, _ := json.Marshal(message)
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

	cleanedDat := cleanedBody{CleanedBody: fornresult}

	params := database.CreateChirpParams{
		Body:   cleanedDat.CleanedBody,
		UserID: valid,
	}

	chirp, err := cfg.dbQueries.CreateChirp(r.Context(), params)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		message := broken{
			Error: "Something went wrong",
		}
		errDat, _ := json.Marshal(message)
		w.Write(errDat)
	}

	response := chirpResponse{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	}

	final, _ := json.Marshal(response)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(final))
}

func (cfg *apiConfig) getAllChirps(w http.ResponseWriter, r *http.Request) {
	type chirpResponse struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserID    uuid.UUID `json:"user_id"`
	}

	sortOrder := r.URL.Query().Get("sort")
	if sortOrder == "" {
		sortOrder = "asc"
	}

	sortFn := func(response []chirpResponse) {
		sort.Slice(response, func(i, j int) bool {
			if sortOrder == "desc" {
				return response[i].CreatedAt.After(response[j].CreatedAt)
			}
			return response[i].CreatedAt.Before(response[j].CreatedAt)
		})
	}

	authorIDStr := r.URL.Query().Get("author_id")
	if authorIDStr != "" {
		authorID, err := uuid.Parse(authorIDStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid author_id"))
			return
		}

		chirps, err := cfg.dbQueries.GetChirpByAuthor(r.Context(), authorID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Something went wrong while trying to get all chirps...\n"))
			w.Write([]byte(err.Error()))
			return
		}

		response := make([]chirpResponse, len(chirps))
		for i, chirp := range chirps {
			response[i] = chirpResponse{
				ID:        chirp.ID,
				CreatedAt: chirp.CreatedAt,
				UpdatedAt: chirp.UpdatedAt,
				Body:      chirp.Body,
				UserID:    chirp.UserID,
			}
		}
		sortFn(response)
		final, _ := json.Marshal(response)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(final)
		return
	}

	chirps, err := cfg.dbQueries.GetAllChirps(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong while trying to get all chirps...\n"))
		w.Write([]byte(err.Error()))
	}

	response := make([]chirpResponse, len(chirps))
	for i, chirp := range chirps {
		response[i] = chirpResponse{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		}
	}
	sortFn(response)
	final, err := json.Marshal(response)
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(final)
}

func (cfg *apiConfig) getChirpById(w http.ResponseWriter, r *http.Request) {
	type chirpResponse struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserID    uuid.UUID `json:"user_id"`
	}

	chirpIDStr := r.PathValue("chirpID")
	ChirpID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		http.Error(w, "Invalid chirp ID", http.StatusNotFound)
		return
	}

	ok, err := cfg.dbQueries.QueryChirpById(r.Context(), ChirpID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Something went wrong"))
		w.Write([]byte(err.Error()))
	}

	response := chirpResponse{
		ID:        ok.ID,
		CreatedAt: ok.CreatedAt,
		UpdatedAt: ok.UpdatedAt,
		Body:      ok.Body,
		UserID:    ok.UserID,
	}

	final, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong"))
		w.Write([]byte(err.Error()))
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(final)
}

func (cfg *apiConfig) deleteChirp(w http.ResponseWriter, r *http.Request) {
	// Check for token in the header
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("{\"message\": \"No token provided\"}"))
		log.Printf("Error getting JWT, was not provided: %s", err)
		return
	}

	userId, err := auth.ValidateJWT(token, cfg.secretKey)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("{\"message\": \"Invalid token\"}"))
		log.Printf("Error validating JWT: %s", err)
		return
	}

	chirpIDStr := r.PathValue("chirpID")
	ChirpID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Can't find Chirp"))
		w.Write([]byte(err.Error()))
		log.Printf("Error finding chirp with ID: %s", err)
		return
	}

	deleteParams := database.DeleteChirpParams{
		ID:     ChirpID,
		UserID: userId,
	}

	rows, err := cfg.dbQueries.DeleteChirp(r.Context(), deleteParams)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong while trying to delete the chirp...\n"))
		w.Write([]byte(err.Error()))
		log.Printf("Something went wrong while trying to delete the chirp: %s", err)
		return
	}

	if rows == 0 {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("You are not allowed to delete this chirp"))
		log.Printf("User is not allowed to delete this chirp: %s\n", err)
	}

	w.WriteHeader(http.StatusNoContent)
}
