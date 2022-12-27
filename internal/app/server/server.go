package server

import (
	"log"
	"net/http"

	"github.com/valyala/fasthttp"

	"github.com/jon69/shorturl/internal/app/handlers"
)

func RunNetHTTP() {
	handler1 := handlers.MakeMyHandler()

	server := &http.Server{
		Addr:    "localhost:8080",
		Handler: handler1,
	}
	log.Fatal(server.ListenAndServe())
}

//----------------------------------------------
func RunFastHTTP() {
	handler1 := handlers.MakeMyHandlerFastHTTP()
	fasthttp.ListenAndServe("localhost:8080", handler1.HandleFastHTTP)
}
