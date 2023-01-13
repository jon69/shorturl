package main

import (
	"log"
	"os"

	"github.com/jon69/shorturl/internal/app/server"
)

func main() {
	server_address := os.Getenv("SERVER_ADDRESS")
	base_url := os.Getenv("BASE_URL")

	log.Print("os SERVER_ADDRESS=" + server_address)
	log.Print("os BASE_URL=" + base_url)

	if server_address == "" {
		server_address = "127.0.0.1:8080"
	}
	if base_url == "" {
		base_url = "http://localhost:8080/"
	}
	server.RunNetHTTP(server_address, base_url)
}
