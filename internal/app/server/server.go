package server

import (
	"log"
	"net/http"

	"compress/gzip"
	"io"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jon69/shorturl/internal/app/handlers"
)

func RunNetHTTP(serverAddress string, baseURL string, filePath string) {
	handler := handlers.MakeMyHandler(baseURL, filePath)

	r := chi.NewRouter()

	r.Get("/{id}", gzipHandle(handler.ServeGetHTTP))
	r.Post("/", gzipHandle(handler.ServePostHTTP))
	r.Post("/api/shorten", gzipHandle(handler.ServeShortenPostHTTP))

	log.Fatal(http.ListenAndServe(serverAddress, r))
}

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	// w.Writer будет отвечать за gzip-сжатие, поэтому пишем в него
	log.Print("compressing data...", b)
	return w.Writer.Write(b)
}

func gzipHandle(nextFunc http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.Header.Get("Content-Encoding") == `gzip` {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			r.Body = gz
		}

		// проверяем, что клиент поддерживает gzip-сжатие
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			// если gzip не поддерживается, передаём управление
			// дальше без изменений
			nextFunc(w, r)
			return
		}

		// создаём gzip.Writer поверх текущего w
		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer gz.Close()

		w.Header().Set("Content-Encoding", "gzip")
		// передаём обработчику страницы переменную типа gzipWriter для вывода данных
		nextFunc(gzipWriter{ResponseWriter: w, Writer: gz}, r)
	})
}
