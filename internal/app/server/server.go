package server

import (
	"log"
	"net/http"

	"github.com/jon69/shorturl/internal/app/handlers"
)

func Run() {
	handler1 := handlers.MakeMyHandler()

	server := &http.Server{
		Addr:    "localhost:8080",
		Handler: handler1,
	}
	log.Fatal(server.ListenAndServe())
}
