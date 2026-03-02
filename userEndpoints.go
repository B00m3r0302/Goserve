package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/B00m3r0302/Goserve/internal/auth"
	"github.com/B00m3r0302/Goserve/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) createUser(w http.ResponseWriter, r *http.Request) {
	type data struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type createdUser struct {
		ID          uuid.UUID `json:"id"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		Email       string    `json:"email"`
		IsChirpyRed bool      `json:"is_chirpy_red"`
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
		ID:          newUser.ID,
		CreatedAt:   newUser.CreatedAt,
		UpdatedAt:   newUser.UpdatedAt,
		Email:       newUser.Email,
		IsChirpyRed: newUser.IsChirpyRed,
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
		IsChirpyRed  bool      `json:"is_chirpy_red"`
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
		IsChirpyRed:  user.IsChirpyRed,
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

func (cfg *apiConfig) updateUser(w http.ResponseWriter, r *http.Request) {
	type input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type returnUser struct {
		ID          uuid.UUID `json:"id"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		Email       string    `json:"email"`
		IsChirpyRed bool      `json:"is_chirpy_red"`
	}

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
	}

	user, err := cfg.dbQueries.GetUserById(r.Context(), userId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong while trying to get the user...\n"))
		w.Write([]byte(err.Error()))
		log.Printf("Something went wrong while trying to get the user: %s", err)
		return
	}

	decoder := json.NewDecoder(r.Body)
	dat := input{}
	err = decoder.Decode(&dat)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		message := map[string]string{"error": "could not decode JSON body"}
		errDat, _ := json.Marshal(message)
		w.Write(errDat)
		log.Printf("Error decoding JSON body: %s", err)
		return
	}

	hashedPassword, err := auth.HashPassword(dat.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong while trying to hash the password...\n"))
		w.Write([]byte(err.Error()))
		log.Printf("Something went wrong while trying to hash the password: %s", err)
		return
	}

	updatedInfo := database.UpdateUserDataParams{
		ID:             user.ID,
		Email:          dat.Email,
		HashedPassword: hashedPassword,
	}

	err = cfg.dbQueries.UpdateUserData(r.Context(), updatedInfo)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong while trying to update the user...\n"))
		w.Write([]byte(err.Error()))
		log.Printf("Something went wrong while trying to update the user: %s", err)
		return
	}

	finalStruct := returnUser{
		ID:          user.ID,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   time.Now(),
		Email:       dat.Email,
		IsChirpyRed: user.IsChirpyRed,
	}

	finalResponse, err := json.Marshal(finalStruct)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong while trying to marshal the user...\n"))
		w.Write([]byte(err.Error()))
		log.Printf("Something went wrong while trying to marshal the user: %s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(finalResponse)
}

func (cfg *apiConfig) upgradeToRed(w http.ResponseWriter, r *http.Request) {
	type data struct {
		UserID uuid.UUID `json:"user_id"`
	}

	type input struct {
		Event string `json:"event"`
		Data  data   `json:"data"`
	}

	// Check for token in the header
	ApiKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("{\"message\": \"No API Key provided\"}"))
		log.Printf("Error getting API Key, was not provided: %s", err)
		return
	}

	if ApiKey != cfg.PolkaAPIKey {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("{\"message\": \"Invalid API Key\"}"))
		log.Printf("Error validating API Key: %s", err)
	}

	decoder := json.NewDecoder(r.Body)
	dat := input{}
	err = decoder.Decode(&dat)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		message := map[string]string{"error": "could not decode JSON body"}
		errDat, _ := json.Marshal(message)
		w.Write(errDat)
		log.Printf("Error decoding JSON body: %s", err)
		return
	}

	if dat.Event != "user.upgraded" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	err = cfg.dbQueries.UpgradeToRed(r.Context(), dat.Data.UserID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
