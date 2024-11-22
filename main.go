package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/MatthewTully/simple-chat-server/internal/client"
	"github.com/MatthewTully/simple-chat-server/internal/server"
	"github.com/joho/godotenv"
)

var portArg int
var hostModeArg bool

func main() {
	godotenv.Load()

	flag.IntVar(&portArg, "port", 0, "Define the port for the server to listen on")
	flag.IntVar(&portArg, "p", 0, "Define the port for the server to listen on (shorthand)")
	flag.BoolVar(&hostModeArg, "host", false, "Launch application as a server host.")
	flag.BoolVar(&hostModeArg, "h", false, "Launch application as a server host.")

	flag.Parse()

	cfg := client.ClientConfig{
		Username:   "Tully",
		UserColour: "purple",
	}

	cli := client.NewClient(&cfg)

	if hostModeArg {
		var port string

		historySize, err := strconv.ParseUint(os.Getenv("SRV_MSG_HISTORY_SIZE"), 10, 64)
		if err != nil {
			log.Fatalf("could not parse History size to uint: %v", err)
		}

		maxConnectionLimit, err := strconv.ParseUint(os.Getenv("SRV_MAX_CONNECTIONS"), 10, 64)
		if err != nil {
			log.Fatalf("could not parse Max connection limit to uint: %v", err)
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
		srv.MaxConnectionLimit = uint(maxConnectionLimit)
		if err != nil {
			log.Fatalln(err)
		}

		go srv.StartListening()
		cli.Connect(fmt.Sprintf("127.0.0.1:%s", port))
	}

	//TODO Remove tmp setup of manual config.

	go cli.AwaitMessage()
	client.StartTUI(&cli)
}
