// Модуль handlers осуществляет обработку конкретных HTTP запросов.
package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	dbh "github.com/jon69/shorturl/internal/app/db"
	"github.com/jon69/shorturl/internal/app/storage"
)

// CTXKey структура для хранения конекста HTTP запроса с информацией о польльзователе.
type CTXKey struct {
}

// MyHandler хранит информацию об обработчике.
type MyHandler struct {
	// urlstorage - хранилище данных.
	urlstorage *storage.StorageURL
	// baseURL - адрес (хост:порт) для выдачи сохраненных URL.
	baseURL string
	// conndb - параметры подключения к БД.
	conndb string
	// trustedSubNet - доверенная подсеть.
	trustedSubNet string
	// ipnet - подсеть
	ipnet *net.IPNet
}

// MyHandler созает новый обработчик.
func MakeMyHandler(filePath string, conndb string) MyHandler {
	h := MyHandler{}
	h.urlstorage = storage.NewStorage(filePath, conndb)
	h.conndb = conndb
	h.trustedSubNet = ""
	return h
}

// SetBaseURL устанавливает новое значение адреса запуска сервера обработки HTTP запросов.
func (h *MyHandler) SetBaseURL(url string) {
	h.baseURL = url
}

// SetTrustedSubNet устанавливает доверенную подсеть.
func (h *MyHandler) SetTrustedSubNet(str string) {
	if str == "" {
		return
	}
	_, ipnet, err := net.ParseCIDR(str)
	if err != nil {
		log.Print("error parse CIDR: " + err.Error())
		return
	}
	h.trustedSubNet = str
	h.ipnet = ipnet
}

// ServeGetPING обрабатывает запрос на проверку подключения к БДServeGetPING
func (h *MyHandler) ServeGetPING(w http.ResponseWriter, r *http.Request) {
	log.Println("ServeGetPING")
	if h.conndb != "" {
		if dbh.Ping(h.conndb) {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

// ServeGetHTTP обрабатывает GET запрос за получение полного URL по его краткой формте.
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

// ServeGetAllURLS обрабатывает GET запрос за получение всех URL.
func (h *MyHandler) ServeGetAllURLS(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var urlsJSON []byte
	var ok bool

	if v := ctx.Value(CTXKey{}); v != nil {
		_, urlsJSON, ok = h.urlstorage.GetUserURLS(fmt.Sprintf("%v", v), h.baseURL)
	} else {
		_, urlsJSON, ok = h.urlstorage.GetURLS(h.baseURL)
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

// ServeGetStats обрабатывает GET запрос за получение статистики.
func (h *MyHandler) ServeGetStats(w http.ResponseWriter, r *http.Request) {
	realIP := r.Header.Get("X-Real-IP")
	ip := net.ParseIP(realIP)

	if h.trustedSubNet == "" || ip == nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if !h.ipnet.Contains(ip) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	statJSON, ok := h.urlstorage.GetStat()

	if !ok {
		http.Error(w, "can not get stat", http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(statJSON)
}

// ServePostHTTP обрабатывает POST запрос на сохранение нового URL.
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

// MyURL хранит информацию о URL.
type MyURL struct {
	// URL -  URL в формате JSON
	URL string `json:"url"`
}

// MyURL хранит информацию о URL для выдачи.
type MyResultURL struct {
	// URL -  URL в формате JSON
	URL string `json:"result"`
}

// ServePostHTTP обрабатывает POST запрос на сохранение нового URL в формате JSON.
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

// MyBatchURL хранит информацию о множестве URL.
type MyBatchURL struct {
	// OriginalURL - исходный URL в формате JSON.
	OriginalURL string `json:"original_url"`
	// CorrelationID - идентификатор соответсвующего URL в формате JSON.
	CorrelationID string `json:"correlation_id"`
}

// MyBatchURL хранит информацию о множестве URL для выдачи пользователю.
type MyBatchResultURL struct {
	// ShortURL - краткая формате URL в формате JSON
	ShortURL string `json:"short_url"`
	// CorrelationID - идентификатор соответсвующего URL в формате JSON.
	CorrelationID string `json:"correlation_id"`
}

// ServePostHTTP обрабатывает POST запрос на сохранение множества новых URL в формате JSON.
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

// MyURLS тип для представлние множетсва URL.
type MyURLS []string

// ServeDeleteBatchHTTP обрабатывает DELETE запрос на удаление можества URL в формате JSON.
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
