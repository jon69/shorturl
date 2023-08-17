package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jon69/shorturl/internal/app/server"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func fillIfEmpty(str string) string {
	if str == "" {
		str = "N/A"
	}
	return str
}

func main() {
	args := os.Args

	fmt.Printf("Build version: %s", fillIfEmpty(buildVersion))
	fmt.Println()
	fmt.Printf("Build date: %s", fillIfEmpty(buildDate))
	fmt.Println()
	fmt.Printf("Build commit: %s", fillIfEmpty(buildCommit))
	fmt.Println()

	log.Printf("Все аргументы запуска: %v\n", args)

	serverAddress := os.Getenv("SERVER_ADDRESS")
	baseURL := os.Getenv("BASE_URL")
	filePath := os.Getenv("FILE_STORAGE_PATH")
	conndb := os.Getenv("DATABASE_DSN")
	enableHTTPS := os.Getenv("ENABLE_HTTPS")

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
	if enableHTTPS == "" {
		flag.StringVar(&enableHTTPS, "s", "", "enable HTTPS")
	}

	flag.Parse()

	serv := server.MakeMyServer()
	serv.SetBaseURL(baseURL)
	serv.SetConnDB(conndb)
	serv.SetFilePath(filePath)
	serv.SetServerAddr(serverAddress)
	serv.SetEnableHTTPS(enableHTTPS)

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
