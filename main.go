package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/jeremyu25/go_server/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var (
	profaneWords = map[string]struct{}{
		"kerfuffle": {},
		"sharbert":  {},
		"fornax":    {},
	}
)

type apiConfig struct {
	fileServerHits atomic.Int32
	dbQueries      *database.Queries
	platform       string
}

func (a *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.fileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func main() {
	godotenv.Load()
	dbUrl := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Printf("Error opening DB")
	}
	dbQueries := database.New(db)
	serverMultiplexer := http.ServeMux{}
	server := http.Server{
		Handler: &serverMultiplexer,
		Addr:    ":8080",
	}
	indexHandler := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	apiCfg := apiConfig{
		dbQueries: dbQueries,
		platform:  platform,
	}
	serverMultiplexer.Handle("/app/", apiCfg.middlewareMetricsInc(indexHandler))
	serverMultiplexer.HandleFunc("GET /admin/metrics", apiCfg.getServerHits)
	serverMultiplexer.HandleFunc("POST /admin/reset", apiCfg.resetServerHits)
	serverMultiplexer.HandleFunc("GET /api/healthz", healthEndpoint)
	serverMultiplexer.HandleFunc("POST /api/validate_chirp", validateChirpEndpoint)
	serverMultiplexer.HandleFunc("POST /api/users", apiCfg.createUser)
	fmt.Println("Server is up and listening...")
	server.ListenAndServe()
}

func respondWithError(w http.ResponseWriter, code int, msg string, err error) {
	if err != nil {
		log.Println(err)
	}
	if code > 499 {
		log.Printf("Responding with 5XX Error: %s", msg)
	}
	type errorResponse struct {
		Error string `json:"error"`
	}
	respondWithJson(w, code,
		errorResponse{
			Error: msg,
		})
}

func respondWithJson(w http.ResponseWriter, code int, payload any) {
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(code)
	w.Write(dat)
}
