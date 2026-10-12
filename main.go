package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
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
}

func (a *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.fileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (a *apiConfig) getServerHits(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(200)
	responseString := fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
	</html>`, a.fileServerHits.Load())
	w.Write([]byte(responseString))
}

func (a *apiConfig) resetServerHits(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	a.fileServerHits.Swap(0)
	w.Write([]byte("Reset hits to 0\n"))
}

func main() {
	godotenv.Load()
	dbUrl := os.Getenv("DB_URL")
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
	}
	serverMultiplexer.Handle("/app/", apiCfg.middlewareMetricsInc(indexHandler))
	serverMultiplexer.HandleFunc("GET /api/healthz", healthEndpoint)
	serverMultiplexer.HandleFunc("GET /admin/metrics", apiCfg.getServerHits)
	serverMultiplexer.HandleFunc("POST /admin/reset", apiCfg.resetServerHits)
	serverMultiplexer.HandleFunc("POST /api/validate_chirp", validateChirpEndpoint)
	server.ListenAndServe()
}

func healthEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func validateChirpEndpoint(w http.ResponseWriter, r *http.Request) {
	type requestBody struct {
		Body string `json:"body"`
	}
	type responseBody struct {
		CleanedBody string `json:"cleaned_body"`
	}
	req := requestBody{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding request", err)
		return
	}
	if len(req.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long", nil)
		return
	}
	cleanedWords := cleanWords(req.Body)
	respondWithJson(w, http.StatusOK,
		responseBody{
			CleanedBody: cleanedWords,
		})
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

func cleanWords(rawString string) string {
	cleanedWords := []string{}
	for word := range strings.SplitSeq(rawString, " ") {
		_, found := profaneWords[word]
		if found {
			cleanedWords = append(cleanedWords, "****")
		} else {
			cleanedWords = append(cleanedWords, word)
		}
	}
	return strings.Join(cleanedWords, " ")
}
