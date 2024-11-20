package main

import (
	"log"
	"os"

	"github.com/MatthewTully/simple-chat-server/internal/server"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	port := os.Getenv("SRV_PORT")
	srv, err := server.NewServer(port)
	if err != nil {
		log.Fatalln(err)
	}

	defer srv.Listener.Close()
	srv.StartListening()
}
