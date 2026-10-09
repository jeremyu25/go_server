package main

import (
	"net/http"
)

func main() {
	serverMultiplexer := http.ServeMux{}
	server := http.Server{
		Handler: &serverMultiplexer,
		Addr:    ":8080",
	}
	serverMultiplexer.Handle("/", http.FileServer(http.Dir(".")))
	server.ListenAndServe()
}
