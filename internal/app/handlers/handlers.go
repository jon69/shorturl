package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/jon69/shorturl/internal/app/storage"
)

type MyHandler struct {
	urlstorage *storage.StorageURL
	baseURL    string
}

func MakeMyHandler(baseURL string, filePath string) MyHandler {
	h := MyHandler{}
	h.urlstorage = storage.NewStorage(filePath)
	h.baseURL = baseURL
	return h
}

func (h MyHandler) ServeGetHTTP(w http.ResponseWriter, r *http.Request) {
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
			http.Error(w, "not found "+id, http.StatusNotFound)
		}
		return
	}
	http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
}

func (h MyHandler) ServePostHTTP(w http.ResponseWriter, r *http.Request) {
	log.Print("received request ")
	if r.Method == http.MethodPost {
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
		w.Write([]byte(h.baseURL + "/" + id))
		return
	}
	http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
}

type MyURL struct {
	URL string `json:"url"`
}

type MyResultURL struct {
	URL string `json:"result"`
}

func (h MyHandler) ServeShortenPostHTTP(w http.ResponseWriter, r *http.Request) {
	log.Print("received request ")
	if r.Method == http.MethodPost {
		log.Print("received post method " + r.URL.Path)
		// читаем Body
		b, err := io.ReadAll(r.Body)
		// обрабатываем ошибку
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var murl MyURL
		if err := json.Unmarshal(b, &murl); err != nil {
			log.Print("Unmarshal fail ", err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		url := murl.URL
		log.Print("received body = " + url)
		if url == "" {
			http.Error(w, "empty url in body", http.StatusBadRequest)
			return
		}
		log.Print("url = " + url)
		var mrurl MyResultURL
		mrurl.URL = h.baseURL + "/" + h.urlstorage.PutURL(url)

		txBz, err := json.Marshal(mrurl)
		if err != nil {
			log.Print("Marshal fail ", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(txBz)
		return
	}
	http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
}
