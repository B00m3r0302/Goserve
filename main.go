package main

import (
	_ "github.com/lib/pq"
)

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/B00m3r0302/Goserve/internal/database"
	"github.com/joho/godotenv"
)

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
	mux.HandleFunc("POST /api/chirp")
	mux.HandleFunc("POST /api/users", c.createUser)

	server := &http.Server{
		Addr:    port,
		Handler: mux,
	}

	// Start the server
	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(server.ListenAndServe())
}
