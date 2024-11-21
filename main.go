package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/MatthewTully/simple-chat-server/internal/server"
	"github.com/joho/godotenv"
)

var portArg int

func main() {
	godotenv.Load()

	flag.IntVar(&portArg, "port", 0, "Define the port for the server to listen on")
	flag.IntVar(&portArg, "p", 0, "Define the port for the server to listen on (shorthand)")

	flag.Parse()
	var port string

	historySize, err := strconv.ParseUint(os.Getenv("MSG_HISTORY_SIZE"), 10, 64)
	if err != nil {
		log.Fatalf("could not parse History size to uint: %v", err)
	}

	if portArg == 0 {
		port = os.Getenv("SRV_PORT")
		valid, msg := validatePortString(port)
		if msg != "" {
			fmt.Println(msg)
		}
		if !valid {
			os.Exit(1)
		}
	} else {
		valid, msg := validatePort(portArg)
		if msg != "" {
			fmt.Println(msg)
		}
		if !valid {
			os.Exit(1)
		}
		port = fmt.Sprintf("%d", portArg)
	}

	srv, err := server.NewServer(port, uint(historySize))
	if err != nil {
		log.Fatalln(err)
	}

	defer srv.Listener.Close()
	srv.StartListening()
}
