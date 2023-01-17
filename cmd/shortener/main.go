package main

import (
	"log"
	"os"

	"github.com/jon69/shorturl/internal/app/server"
)

func main() {
	serverAddress := os.Getenv("SERVER_ADDRESS")
	baseURL := os.Getenv("BASE_URL")
	filePath := os.Getenv("FILE_STORAGE_PATH")

	log.Print("os FILE_STORAGE_PATH=" + filePath)
	log.Print("os SERVER_ADDRESS=" + serverAddress)
	log.Print("os BASE_URL=" + baseURL)

	if serverAddress == "" {
		serverAddress = "127.0.0.1:8080"
	}
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	server.RunNetHTTP(serverAddress, baseURL, filePath)
}
