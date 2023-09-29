package example_test

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/jon69/shorturl/internal/app/handlers"
	"github.com/jon69/shorturl/internal/app/storage"
)

func Example() {
	// создаем обработчик
	urlstorage := storage.NewStorage("", "")
	handler := handlers.MakeMyHandler("", urlstorage)
	r := chi.NewRouter()

	r.Get("/ping", handler.ServeGetPING)
	r.Get("/{id}", handler.ServeGetHTTP)
	r.Get("/api/user/urls", handler.ServeGetAllURLS)
	r.Post("/", handler.ServePostHTTP)
	r.Post("/api/shorten", handler.ServeShortenPostHTTP)
	r.Post("/api/shorten/batch", handler.ServeShortenPostBatchHTTP)
	r.Delete("/api/user/urls", handler.ServeDeleteBatchHTTP)
	// запускаем сервер по обработке HTTP запросов
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", r))
}
