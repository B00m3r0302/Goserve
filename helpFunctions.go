package main

import (
	_ "github.com/lib/pq"
)

import (
	"fmt"
	"net/http"
)

func (cfg *apiConfig) handlerChirpsGet(w http.ResponseWriter, r *http.Request) {
	chirpIDStr := r.PathValue("chirpID")
	// parse UUID -> query DB -> 200 or 404
}
