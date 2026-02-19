package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
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

	good := valid{
		Valid: true,
	}

	goodDat, err := json.Marshal(good)
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(goodDat)
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
	// Create a new mux and server struct
	filepathRoot := "."
	port := ":8080"
	healthHandler := http.HandlerFunc(healthCheck)
	c := &apiConfig{}
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
