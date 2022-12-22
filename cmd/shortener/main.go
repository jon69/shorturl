package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
)

var counter uint64 = 0

func getNewId() string {
	for {
		val := atomic.LoadUint64(&counter)
		if atomic.CompareAndSwapUint64(&counter, val, val+1) {
			str := fmt.Sprint(counter)
			return str
		}
	}
}

type MyHandler struct {
	shorturlMap map[string]string
	mux         *sync.RWMutex
}

func (h MyHandler) put(key string, value string) {
	h.mux.Lock()
	h.shorturlMap[key] = value
	h.mux.Unlock()
}

func (h MyHandler) get(id string) (string, bool) {
	h.mux.RLock()
	val, ok := h.shorturlMap[id]
	h.mux.RUnlock()
	return val, ok
}

func (h MyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodGet {

		id := r.URL.Path[1:]
		if id == "" {
			http.Error(w, "The query parameter is missing", http.StatusBadRequest)
			return
		}
		log.Print("found id = " + id)
		val, ok := h.get(id)
		if ok {
			w.Header().Set("Location", val)
			w.WriteHeader(http.StatusTemporaryRedirect)
		} else {
			http.Error(w, "not found "+id, http.StatusBadRequest)
		}
		return

	} else if r.Method == http.MethodPost {
		// читаем Body
		b, err := io.ReadAll(r.Body)
		// обрабатываем ошибку
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		url := string(b)
		if url == "" {
			http.Error(w, "empty url in body", http.StatusBadRequest)
			return
		}
		log.Print("url = " + url)
		id := getNewId()
		log.Print("id = " + id)
		h.put(id, url)
		w.Header().Set("content-type", "plain/text")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(id))
		return
	}
	http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	return
}

func main() {
	var handler1 MyHandler

	handler1.shorturlMap = make(map[string]string)
	handler1.mux = &sync.RWMutex{}

	server := &http.Server{
		Addr:    "localhost:8080",
		Handler: handler1,
	}
	log.Fatal(server.ListenAndServe())
}
