package server

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jon69/shorturl/internal/app/handlers"
)

func RunNetHTTP(serverAddress string, baseURL string) {
	handler := handlers.MakeMyHandler(baseURL)

	r := chi.NewRouter()

	r.Get("/{id}", handler.ServeGetHTTP)
	r.Post("/", handler.ServePostHTTP)
	r.Post("/api/shorten", handler.ServeShortenPostHTTP)

	log.Fatal(http.ListenAndServe(serverAddress, r))
}
