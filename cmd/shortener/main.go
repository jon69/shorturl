package main

import (
	"crypto/rand"
	"flag"
	"log"
	"os"

	"github.com/jon69/shorturl/internal/app/server"
)

func main() {
	args := os.Args
	log.Printf("Все аргументы запуска: %v\n", args)

	serverAddress := os.Getenv("SERVER_ADDRESS")
	baseURL := os.Getenv("BASE_URL")
	filePath := os.Getenv("FILE_STORAGE_PATH")
	conndb := os.Getenv("DATABASE_DSN")

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
	if conndb == "" {
		flag.StringVar(&conndb, "d", "", "connection to database")
	}

	flag.Parse()

	serv := server.MakeMyServer()
	serv.SetBaseURL(baseURL)
	serv.SetConnDB(conndb)
	serv.SetFilePath(filePath)
	serv.SetServerAddr(serverAddress)

	key, err := generateRandom(16)
	if err != nil {
		log.Println("error to generate new key")
		return
	}
	log.Println("generated new key")
	serv.SetSecretKey(key)
	serv.RunNetHTTP()
}

func generateRandom(size int) ([]byte, error) {
	// генерируем случайную последовательность байт
	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}
