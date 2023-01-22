package main

import (
	"flag"
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
		flag.StringVar(&serverAddress, "a", "127.0.0.1:8080", "server address")
	}
	if baseURL == "" {
		flag.StringVar(&baseURL, "b", "http://127.0.0.1:8080", "base url")
	}
	if filePath == "" {
		flag.StringVar(&filePath, "f", "", "path to file")
	}
	log.Print("path to file = " + filePath)
	log.Print("server address = " + serverAddress)
	log.Print("base url = " + baseURL)

	server.RunNetHTTP(serverAddress, baseURL, filePath)
}
