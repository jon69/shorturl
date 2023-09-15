// Модуль server представляет абстрацию сервера по обработке запросов.
package server

import (
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	_ "net/http/pprof"

	"github.com/go-chi/chi/v5"
	uuid "github.com/satori/go.uuid"

	rpcsrv "github.com/jon69/shorturl/internal/app/grpcserver"
	"github.com/jon69/shorturl/internal/app/handlers"
	"github.com/jon69/shorturl/internal/app/httpsmaker"
	"github.com/jon69/shorturl/internal/app/storage"
)

// MyServer хранит информацию о сервере.
type MyServer struct {
	// serverAddress - адрес (хост:порт) по которому запускается сервер.
	serverAddress string
	// baseURL - адрес (хост:порт) для выдачи сохраненных URL.
	baseURL string
	// filePath - путь до файла с информацией о сохраненных URL.
	filePath string
	// key - секретный ключ на подписи куки.
	key []byte
	// conndb - параметры подключения к БД.
	conndb string
	// enableHTTPS - признак использования HTTPS.
	enableHTTPS bool
	// trustedSubNet - доверенная подсеть.
	trustedSubNet string
}

// MakeMyServer создает новый сервер.
func MakeMyServer() MyServer {
	h := MyServer{}
	h.enableHTTPS = false
	return h
}

// SetServerAddr устанавливает новое значение адреса на котором запускается сервер.
func (h *MyServer) SetServerAddr(str string) {
	h.serverAddress = str
	log.Print("server address=" + h.serverAddress)
}

// SetBaseURL устанавливает новое значение адреса для выдачи сохраненных URL.
func (h *MyServer) SetBaseURL(str string) {
	h.baseURL = str
	log.Print("base url=" + h.baseURL)
}

// SetFilePath устанавливает новое значение пути для сохранение URL.
func (h *MyServer) SetFilePath(str string) {
	h.filePath = str
	log.Print("path to file=" + h.filePath)
}

// SetSecretKey устанавливает новое значение секретного ключа.
func (h *MyServer) SetSecretKey(b []byte) {
	h.key = b
}

// SetConnDB устанавливает новое значение параметров подключения к БД.
func (h *MyServer) SetConnDB(str string) {
	h.conndb = str
	log.Print("connection to db=" + h.conndb)
}

// SetTrustedSubNet устанавливает значение доверенной подсети.
func (h *MyServer) SetTrustedSubNet(str string) {
	h.trustedSubNet = str
	log.Print("trustedSubNet=" + h.trustedSubNet)
}

// SetEnableHTTPS устанавливает признак использования HTTPS соединения.
func (h *MyServer) SetEnableHTTPS(str string) {
	if str != "" {
		h.enableHTTPS = true
	} else {
		h.enableHTTPS = false
	}
	log.Print("enable HTTPS=" + str)
}

// RunServers устанавливает обработчки и запускает сервера.
func (h *MyServer) RunServers() {

	sigs := make(chan os.Signal, 1)
	// регистрируем перенаправление прерываний
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	// создаем потокобезопасное хранилище общее для HTTP и gRPC
	urlstorage := storage.NewStorage(h.filePath, h.conndb)

	// создаем gRPC сервер для обработки
	rpcServer := rpcsrv.MakeServer(h.baseURL, h.conndb, urlstorage)

	// создаем HTTP сервер для обработки
	handler := handlers.MakeMyHandler(h.conndb, urlstorage)
	handler.SetBaseURL(h.baseURL)
	handler.SetTrustedSubNet(h.trustedSubNet)
	r := chi.NewRouter()

	r.Get("/ping", handler.ServeGetPING)
	r.Get("/{id}", authHandle(h.key, gzipHandle(handler.ServeGetHTTP)))
	r.Get("/api/user/urls", authHandle(h.key, gzipHandle(handler.ServeGetAllURLS)))
	r.Get("/api/internal/stats", authHandle(h.key, gzipHandle(handler.ServeGetStats)))
	r.Post("/", authHandle(h.key, gzipHandle(handler.ServePostHTTP)))
	r.Post("/api/shorten", authHandle(h.key, gzipHandle(handler.ServeShortenPostHTTP)))
	r.Post("/api/shorten/batch", authHandle(h.key, gzipHandle(handler.ServeShortenPostBatchHTTP)))
	r.Delete("/api/user/urls", authHandle(h.key, gzipHandle(handler.ServeDeleteBatchHTTP)))

	var mainsrv = http.Server{Addr: h.serverAddress, Handler: r}
	var pprofsrv = http.Server{Addr: ":6060"}

	// запускаем горутину обработки пойманных прерываний
	go func() {
		// читаем из канала прерываний
		<-sigs
		log.Println("interrupted...graceful shutdown")
		// завершаем работу PRC севера
		rpcServer.Shutdown()
		// получили сигнал запускаем процедуру graceful shutdown
		if err := pprofsrv.Shutdown(context.Background()); err != nil {
			log.Printf("Pprof HTTP server Shutdown: %v", err)
		}
		// получили сигнал запускаем процедуру graceful shutdown
		if err := mainsrv.Shutdown(context.Background()); err != nil {
			log.Printf("Main HTTP server Shutdown: %v", err)
		}
	}()

	go func() {
		err := rpcServer.Serve()
		if err != nil {
			log.Printf("rpcServer Serve exited with err: %v", err)
		} else {
			log.Printf("rpcServer Serve exited.")
		}
	}()

	// только для pprof приходится запустить отдельный сервер
	go func() {
		err := pprofsrv.ListenAndServe()
		if err != nil {
			log.Printf("pprofsrv ListenAndServe exited with err: %v", err)
		} else {
			log.Printf("pprofsrv ListenAndServe exited.")
		}
	}()

	if h.enableHTTPS {
		certFile, keyFile, err := httpsmaker.MakeHTTPS()
		if err != nil {
			log.Fatal(err)
		}
		err = mainsrv.ListenAndServeTLS(certFile, keyFile)
		if err != nil {
			log.Printf("mainsrv ListenAndServeTLS exited with err: %v", err)
		} else {
			log.Printf("mainsrv ListenAndServeTLS exited.")
		}
	} else {
		err := mainsrv.ListenAndServe()
		if err != nil {
			log.Printf("mainsrv ListenAndServe exited with err: %v", err)
		} else {
			log.Printf("mainsrv ListenAndServe exited.")
		}
	}
}

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

// Write записывает данные в сжатом формате
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

func getNewSignedCookie(secretKey []byte) (http.Cookie, string) {
	// создаем новый идентификатор
	myuuid := uuid.NewV4()
	log.Println("new UUID is: ", myuuid.String())

	cookieUID := http.Cookie{
		Name:  "uid",
		Value: myuuid.String(),
	}

	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(cookieUID.Name))
	mac.Write([]byte(cookieUID.Value))
	signature := mac.Sum(nil)
	signatureHEX := hex.EncodeToString(signature)
	// Prepend the cookie value with the HMAC signature.
	cookieUID.Value = signatureHEX + "-" + cookieUID.Value
	return cookieUID, myuuid.String()
}

func validateCookie(secretKey []byte, cookieSigned *http.Cookie) (bool, string) {
	// A SHA256 HMAC signature has a fixed length of 32 bytes. To avoid a potential
	// 'index out of range' panic in the next step, we need to check sure that the
	// length of the signed cookie value is at least this long. We'll use the
	// sha256.Size constant here, rather than 32, just because it makes our code
	// a bit more understandable at a glance.
	signedValue := cookieSigned.Value
	if len(signedValue) < 4 {
		return false, ""
	}

	i := strings.Index(signedValue, "-")
	if i == -1 {
		log.Println("not found: - ")
		return false, ""
	}

	log.Println("i=", i)

	// Split apart the signature and original cookie value.
	signatureHEX := signedValue[:i]
	value := signedValue[i+1:]

	log.Println("signature=", signatureHEX)
	log.Println("value=", value)

	signature, err := hex.DecodeString(signatureHEX)
	if err != nil {
		log.Println("error to decode hex signature")
		return false, ""
	}

	// Recalculate the HMAC signature of the cookie name and original value.
	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(cookieSigned.Name))
	mac.Write([]byte(value))
	expectedSignature := mac.Sum(nil)

	// Check that the recalculated signature matches the signature we received
	// in the cookie. If they match, we can be confident that the cookie name
	// and value haven't been edited by the client.
	if !hmac.Equal(signature, expectedSignature) {
		log.Println("cookie not equal")
		return false, ""
	}
	log.Println("cookie equal")
	// Return the original cookie value.
	return true, value
}

func authHandle(secretKey []byte, nextFunc http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Print("received request, method = ", r.Method)
		log.Print("request, path = ", r.URL.Path)
		uid := "emptyString"
		uidCookie, err := r.Cookie("uid")
		if err != nil { // куки нет, либо ошибка
			switch {
			case errors.Is(err, http.ErrNoCookie): // куки нет
				log.Println("cookie not found")
				log.Println(err)
				var newCookie http.Cookie
				newCookie, uid = getNewSignedCookie(secretKey)
				http.SetCookie(w, &newCookie)
			default: // ошибка
				log.Println(err)
				http.Error(w, "server error", http.StatusInternalServerError)
				return
			}
		} else { // кука есть
			log.Println("found cookie=" + uidCookie.Value)
			var equal bool
			equal, uid = validateCookie(secretKey, uidCookie)
			if !equal {
				var newCookie http.Cookie
				newCookie, uid = getNewSignedCookie(secretKey)
				http.SetCookie(w, &newCookie)
			}
		}
		ctx := r.Context()
		ctx = context.WithValue(ctx, handlers.CTXKey{}, uid)
		nextFunc(w, r.WithContext(ctx))
	})
}
