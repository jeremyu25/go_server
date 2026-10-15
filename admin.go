package main

import (
	"fmt"
	"net/http"
)

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
	if a.platform != "dev" {
		respondWithError(w, http.StatusForbidden, "Current platform is not dev, cannot reset user database", nil)
	}
	if err := a.dbQueries.DeleteAllUsers(r.Context()); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error resetting users database", err)
	}
}
