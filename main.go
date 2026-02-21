package main

import _ "github.com/lib/pq"

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"sync/atomic"

	"github.com/B00m3r0302/Goserve/internal/database"
	"github.com/joho/godotenv"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func createUser(w http.ResponseWriter, r *http.Request) {
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
	}

}

func validateChirp(w http.ResponseWriter, r *http.Request) {
	type data struct {
		Body string `json:"body"`
	}

	type broken struct {
		Error string `json:"error"`
	}

	type valid struct {
		Valid bool `json:"valid"`
	}

	decoder := json.NewDecoder(r.Body)
	dat := data{}
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
	cfg.fileserverHits.Store(0)
	w.WriteHeader(http.StatusOK)
}

func main() {
	// Load env variables
	godotenv.Load()

	// Set variables
	dbURL := os.Getenv("DB_URL")

	// Open a connection to the database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create a new mux and server struct
	dbQueries := database.New(db)
	filepathRoot := "."
	port := ":8080"
	healthHandler := http.HandlerFunc(healthCheck)
	c := &apiConfig{
		dbQueries: dbQueries,
	}
	mux := http.NewServeMux()
	mux.Handle("/app/", c.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(filepathRoot)))))
	mux.Handle("GET /api/healthz", healthHandler)
	mux.HandleFunc("GET /admin/metrics", c.hitsCount)
	mux.HandleFunc("POST /admin/reset", c.hitsReset)
	mux.HandleFunc("POST /api/validate_chirp", validateChirp)

	server := &http.Server{
		Addr:    port,
		Handler: mux,
	}

	// Start the server
	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(server.ListenAndServe())
}
