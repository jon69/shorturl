package handlers

import (
	"io"
	"log"
	"net/http"

	"github.com/jon69/shorturl/internal/app/storage"
)

type MyHandler struct {
	urlstorage *storage.StorageURL
}

func MakeMyHandler() MyHandler {
	h := MyHandler{}
	h.urlstorage = storage.NewStorage()
	return h
}

func (h MyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Print("received request ")
	if r.Method == http.MethodGet {
		log.Print("received get method " + r.URL.Path)
		id := r.URL.Path[1:]
		if id == "" {
			http.Error(w, "The query parameter is missing", http.StatusBadRequest)
			return
		}
		log.Print("parsed id = " + id)
		val, ok := h.urlstorage.GetURL(id)
		if ok {
			log.Print("found value = " + val)
			w.Header().Set("Location", val)
			w.WriteHeader(http.StatusTemporaryRedirect)
		} else {
			log.Print("not found id " + id)
			http.Error(w, "not found "+id, http.StatusBadRequest)
		}
		return

	} else if r.Method == http.MethodPost {
		log.Print("received post method " + r.URL.Path)
		// читаем Body
		b, err := io.ReadAll(r.Body)
		// обрабатываем ошибку
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		url := string(b)
		log.Print("received body = " + url)
		if url == "" {
			http.Error(w, "empty url in body", http.StatusBadRequest)
			return
		}
		log.Print("url = " + url)
		id := h.urlstorage.PutURL(url)
		w.Header().Set("content-type", "plain/text")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("http://localhost:8080/" + id))
		return
	}
	http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
}
