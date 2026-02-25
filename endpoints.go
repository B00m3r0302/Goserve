package main

import (
	"log"
	"time"

	"github.com/B00m3r0302/Goserve/internal/auth"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"

	"github.com/B00m3r0302/Goserve/internal/database"
)

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) createUser(w http.ResponseWriter, r *http.Request) {
	type data struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type createdUser struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	dat := data{}
	err := decoder.Decode(&dat)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		message := map[string]string{"error": "could not decode JSON body"}
		errDat, _ := json.Marshal(message)
		w.Write(errDat)
		return
	}

	hashedPassword, err := auth.HashPassword(dat.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong while hashing the password...\n"))
		w.Write([]byte(err.Error()))

		log.Println(err)
	}

	newUserParams := database.InsertUserParams{
		Email:          dat.Email,
		HashedPassword: hashedPassword,
	}

	err = cfg.dbQueries.InsertUser(r.Context(), newUserParams)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		message := map[string]string{"error": "Something went wrong while inserting the user into the database"}
		errDat, _ := json.Marshal(message)
		w.Write(errDat)
		log.Println(err)
		return
	}

	newUser, err := cfg.dbQueries.LookupUserByEmail(r.Context(), dat.Email)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong while trying to get the user by email...\n"))
		w.Write([]byte(err.Error()))
	}

	response := createdUser{
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

func (cfg *apiConfig) loginUser(w http.ResponseWriter, r *http.Request) {
	type data struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type createdUser struct {
		ID           uuid.UUID `json:"id"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		Email        string    `json:"email"`
		Token        string    `json:"token"`
		RefreshToken string    `json:"refresh_token"`
	}

	decoder := json.NewDecoder(r.Body)
	dat := data{}
	err := decoder.Decode(&dat)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		message := map[string]string{"error": "could not decode JSON body"}
		errDat, _ := json.Marshal(message)
		w.Write(errDat)
		return
	}

	// Check password hash
	user, err := cfg.dbQueries.LookupUserByEmail(r.Context(), dat.Email)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong while trying to get the users password...\n"))
		w.Write([]byte(err.Error()))
		return
	}

	match, err := auth.CheckPasswordHash(dat.Password, user.HashedPassword)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong while trying to check the password...\n"))
		return
	}

	// Generate JWT
	jwt, err := auth.MakeJWT(user.ID, cfg.secretKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong while trying to generate the JWT...\n"))
		w.Write([]byte(err.Error()))
		return
	}

	// Refresh Token
	refreshToken := auth.MakeRefreshToken()

	// Store refresh token in database
	addRefreshTokenParams := database.AddRefreshTokenParams{
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Hour * 24 * 60),
	}

	err = cfg.dbQueries.AddRefreshToken(r.Context(), addRefreshTokenParams)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong while trying to store the refresh token...\n"))
		w.Write([]byte(err.Error()))
		log.Println(err)
		return
	}

	loggedInUser := createdUser{
		ID:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Email:        user.Email,
		Token:        jwt,
		RefreshToken: refreshToken,
	}
	response, err := json.Marshal(loggedInUser)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong while trying to marshal the user...\n"))
		return
	}

	if match == true {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(response)

	} else {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("{\"message\": \"Login failed\"}"))
	}

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
		log.Println("Error getting JWT: %s", err)
		return
	}

	valid, err := auth.ValidateJWT(jwt, cfg.secretKey)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("{\"message\": \"Unauthorized\"}"))
		log.Println("Error validating JWT: %s", err)
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

	chirps, err := cfg.dbQueries.GetAllChirps(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong while trying to get all chirps...\n"))
		w.Write([]byte(err.Error()))
	}

	response := make([]chirpResponse, len(chirps))
	for chirp := range chirps {
		response[chirp] = chirpResponse{
			ID:        chirps[chirp].ID,
			CreatedAt: chirps[chirp].CreatedAt,
			UpdatedAt: chirps[chirp].UpdatedAt,
			Body:      chirps[chirp].Body,
			UserID:    chirps[chirp].UserID,
		}
	}
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
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(final)
}

func (cfg *apiConfig) refreshServer(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("{\"message\": \"No token provided\"}"))
		log.Println("Error getting JWT, was not provided: %s", err)
		return
	}

	_, err = auth.ValidateJWT(token, cfg.secretKey)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("{\"message\": \"Header is not valid\"}"))
		log.Println("Error validating JWT INVALID: %s", err)
		return
	}

	err = cfg.dbQueries.CheckTokenExpired(r.Context(), token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("{\"message\": \"Token is expired\"}"))
		log.Println("Error validating JWT EXPIRED: %s", err)
	}

	w.WriteHeader(http.StatusOK)

}
