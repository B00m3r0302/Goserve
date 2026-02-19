package main

import (
	"log"
	"net/http"
)

func main() {
	// Create a new mux and server struct
	filepathRoot := "."
	port := ":8080"
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(filepathRoot)))
	server := &http.Server{
		Addr:    port,
		Handler: mux,
	}

	// Start the server
	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(server.ListenAndServe())
}
