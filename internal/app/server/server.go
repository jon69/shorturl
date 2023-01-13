package server

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jon69/shorturl/internal/app/handlers"
)

func RunNetHTTP(server_address string, base_url string) {
	handler := handlers.MakeMyHandler(base_url)

	r := chi.NewRouter()

	r.Get("/{id}", handler.ServeGetHTTP)
	r.Post("/", handler.ServePostHTTP)
	r.Post("/api/shorten", handler.ServeShortenPostHTTP)

	log.Fatal(http.ListenAndServe(server_address, r))
}
