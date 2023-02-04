package server

import (
	"encoding/hex"
	"log"
	"net/http"

	"compress/gzip"
	"io"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jon69/shorturl/internal/app/handlers"

	"errors"

	"context"
	"crypto/hmac"
	"crypto/sha256"

	uuid "github.com/satori/go.uuid"
)

func RunNetHTTP(serverAddress string, baseURL string, filePath string, key []byte) {
	handler := handlers.MakeMyHandler(baseURL, filePath)

	r := chi.NewRouter()

	r.Get("/{id}", authHandle(key, gzipHandle(handler.ServeGetHTTP)))
	r.Get("/api/user/urls", authHandle(key, gzipHandle(handler.ServeGetAllURLS)))
	r.Post("/", authHandle(key, gzipHandle(handler.ServePostHTTP)))
	r.Post("/api/shorten", authHandle(key, gzipHandle(handler.ServeShortenPostHTTP)))

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
