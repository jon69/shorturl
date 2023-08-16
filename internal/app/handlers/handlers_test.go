package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServeHTTP(t *testing.T) {

	type request struct {
		method string
		url    string
		body   string
	}

	type response struct {
		code        int
		body        string
		contentType string
		location    string
	}

	tests := []struct {
		name string
		req  request
		resp response
	}{
		{
			name: "post",
			req: request{
				method: http.MethodPost,
				url:    "/",
				body:   "http://yandex.ru",
			},
			resp: response{
				code:        201,
				body:        "http://localhost:8080/1",
				contentType: "plain/text",
				location:    "",
			},
		},
		{
			name: "get",
			req: request{
				method: http.MethodGet,
				url:    "/1",
				body:   "",
			},
			resp: response{
				code:        307,
				body:        "",
				contentType: "",
				location:    "http://yandex.ru",
			},
		},
		{
			name: "ping",
			req: request{
				method: http.MethodGet,
				url:    "/ping",
				body:   "",
			},
			resp: response{
				code:        200,
				body:        "",
				contentType: "",
				location:    "",
			},
		},
		{
			name: "post shorten",
			req: request{
				method: http.MethodPost,
				url:    "/api/shorten",
				body:   "{\"url\": \"http://gavrilin.ru\"}",
			},
			resp: response{
				code:        201,
				body:        "{\"result\":\"http://localhost:8080/2\"}",
				contentType: "application/json",
				location:    "",
			},
		},
	}
	hendl := MakeMyHandler("", "")
	hendl.SetBaseURL("http://localhost:8080")

	for _, tt := range tests {
		request := httptest.NewRequest(tt.req.method, tt.req.url, strings.NewReader(tt.req.body))

		// создаём новый Recorder
		w := httptest.NewRecorder()
		// определяем хендлер

		var foo http.HandlerFunc
		if tt.name == "get" {
			foo = hendl.ServeGetHTTP
		} else if tt.name == "post" {
			foo = hendl.ServePostHTTP
		} else if tt.name == "ping" {
			foo = hendl.ServeGetPING
		} else if tt.name == "post shorten" {
			foo = hendl.ServeShortenPostHTTP
		}
		h := http.HandlerFunc(foo)
		// запускаем сервер
		h.ServeHTTP(w, request)
		res := w.Result()

		// проверяем код ответа
		if res.StatusCode != tt.resp.code {
			t.Errorf("Expected status code %d, got %d", tt.resp.code, w.Code)
		}

		// получаем и проверяем тело запроса
		defer res.Body.Close()
		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(resBody) != tt.resp.body {
			t.Errorf("Expected body %s, got %s", tt.resp.body, w.Body.String())
		}
		// заголовок ответа
		if res.Header.Get("Content-Type") != tt.resp.contentType {
			t.Errorf("Expected Content-Type %s, got %s", tt.resp.contentType, res.Header.Get("Content-Type"))
		}

		if res.Header.Get("Location") != tt.resp.location {
			t.Errorf("Expected Location %s, got %s", tt.resp.location, res.Header.Get("Location"))
		}
	}
}
