package server

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jon69/shorturl/internal/app/handlers"
)

func RunNetHTTP() {
	handler1 := handlers.MakeMyHandler()

	r := chi.NewRouter()

	r.Get("/{id}", handler1.ServeHTTP)
	r.Post("/", handler1.ServeHTTP)

	log.Fatal(http.ListenAndServe(":8080", r))
}
