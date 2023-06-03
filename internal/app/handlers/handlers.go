package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"fmt"

	dbh "github.com/jon69/shorturl/internal/app/db"
	"github.com/jon69/shorturl/internal/app/storage"
)

type CTXKey struct {
}

type MyHandler struct {
	urlstorage *storage.StorageURL
	baseURL    string
	conndb     string
}

func MakeMyHandler(filePath string, conndb string) MyHandler {
	h := MyHandler{}
	h.urlstorage = storage.NewStorage(filePath, conndb)
	return h
}

func (h *MyHandler) SetBaseURL(url string) {
	h.baseURL = url
}

func (h *MyHandler) ServeGetPING(w http.ResponseWriter, r *http.Request) {
	log.Println("ServeGetPING")
	if dbh.Ping(h.conndb) {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *MyHandler) ServeGetHTTP(w http.ResponseWriter, r *http.Request) {

	id := r.URL.Path[1:]
	if id == "" {
		http.Error(w, "The query parameter is missing", http.StatusBadRequest)
		return
	}
	log.Print("parsed id = " + id)
	var val string
	var ok bool

	var isDel bool
	val, ok, isDel = h.urlstorage.GetURL(id)

	if ok {
		log.Print("found value = " + val)
		if isDel {
			w.WriteHeader(http.StatusGone)
		} else {
			w.Header().Set("Location", val)
			w.WriteHeader(http.StatusTemporaryRedirect)
		}
	} else {
		log.Print("not found id " + id)
		http.Error(w, "not found "+id, http.StatusNotFound)
	}
}

func (h *MyHandler) ServeGetAllURLS(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var urlsJSON []byte
	var ok bool

	if v := ctx.Value(CTXKey{}); v != nil {
		urlsJSON, ok = h.urlstorage.GetUserURLS(fmt.Sprintf("%v", v), h.baseURL)
	} else {
		urlsJSON, ok = h.urlstorage.GetURLS(h.baseURL)
	}

	if !ok {
		http.Error(w, "cant get all urls", http.StatusInternalServerError)
		return
	}

	if len(urlsJSON) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(urlsJSON)
}

func (h *MyHandler) ServePostHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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

	var id string
	var iou int
	if v := ctx.Value(CTXKey{}); v != nil {
		iou, id = h.urlstorage.PutUserURL(fmt.Sprintf("%v", v), url)
	} else {
		iou, id = h.urlstorage.PutURL(url)
	}

	//id = h.urlstorage.PutURL(url)

	w.Header().Set("content-type", "plain/text")
	if iou == 1 {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusConflict)
	}

	w.Write([]byte(h.baseURL + "/" + id))
}

type MyURL struct {
	URL string `json:"url"`
}

type MyResultURL struct {
	URL string `json:"result"`
}

func (h *MyHandler) ServeShortenPostHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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

	var iou int
	var shortURL string
	if v := ctx.Value(CTXKey{}); v != nil {
		iou, shortURL = h.urlstorage.PutUserURL(fmt.Sprintf("%v", v), url)
	} else {
		iou, shortURL = h.urlstorage.PutURL(url)
	}
	mrurl.URL = h.baseURL + "/" + shortURL

	txBz, err := json.Marshal(mrurl)
	if err != nil {
		log.Print("Marshal fail ", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	if iou == 1 {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusConflict)
	}
	w.Write(txBz)
}

type MyBatchURL struct {
	OriginalURL   string `json:"original_url"`
	CorrelationID string `json:"correlation_id"`
}

type MyBatchResultURL struct {
	ShortURL      string `json:"short_url"`
	CorrelationID string `json:"correlation_id"`
}

func (h *MyHandler) ServeShortenPostBatchHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// читаем Body
	b, err := io.ReadAll(r.Body)
	// обрабатываем ошибку
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var murls []MyBatchURL
	if err := json.Unmarshal(b, &murls); err != nil {
		log.Print("Unmarshal batch fail ", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var iou int
	iou = 1
	var mrurls []MyBatchResultURL
	for _, url := range murls {
		log.Println("received original_url=" + url.OriginalURL + " with correlation_id=" + url.CorrelationID)
		if url.OriginalURL == "" {
			http.Error(w, "empty original_url in body", http.StatusBadRequest)
			return
		}
		var mrurl MyBatchResultURL
		mrurl.CorrelationID = url.CorrelationID

		var iouLocal int
		var shortURL string
		if v := ctx.Value(CTXKey{}); v != nil {
			iouLocal, shortURL = h.urlstorage.PutUserURL(fmt.Sprintf("%v", v), url.OriginalURL)
		} else {
			iouLocal, shortURL = h.urlstorage.PutURL(url.OriginalURL)
		}
		if iou != 2 && iouLocal == 2 {
			iou = 2
		}
		mrurl.ShortURL = h.baseURL + "/" + shortURL
		mrurls = append(mrurls, mrurl)
	}

	txBz, err := json.Marshal(mrurls)
	if err != nil {
		log.Print("Marshal batch urls fail ", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	if iou == 1 {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusConflict)
	}
	w.Write(txBz)
}

type MyURLS []string

func (h *MyHandler) ServeDeleteBatchHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// читаем Body
	b, err := io.ReadAll(r.Body)
	// обрабатываем ошибку
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var urls MyURLS
	if err := json.Unmarshal(b, &urls); err != nil {
		log.Print("while serve delete unmarshal url fail ", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var accepted bool
	accepted = true

	for _, url := range urls {
		log.Println("deleting url=" + url)
		if url == "" {
			http.Error(w, "empty url", http.StatusBadRequest)
			return
		}
		if v := ctx.Value(CTXKey{}); v != nil {
			if !h.urlstorage.DelUserURL(fmt.Sprintf("%v", v), url) {
				accepted = false
				log.Println("can not delete url=", url)
			}
		} else {
			if !h.urlstorage.DelURL(url) {
				accepted = false
				log.Println("can not delete url=", url)
			}
		}
	}
	if accepted {
		w.WriteHeader(http.StatusAccepted)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
