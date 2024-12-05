package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/MatthewTully/simple-chat-server/internal/client"
	"github.com/MatthewTully/simple-chat-server/internal/server"
	"github.com/joho/godotenv"
)

var portArg int
var hostModeArg bool
var setUsrConfArg bool
var logger *log.Logger

func main() {
	godotenv.Load()

	flag.IntVar(&portArg, "port", 0, "Define the port for the server to listen on")
	flag.IntVar(&portArg, "p", 0, "Define the port for the server to listen on (shorthand)")
	flag.BoolVar(&hostModeArg, "host", false, "Launch application as a server host")
	flag.BoolVar(&hostModeArg, "h", false, "Launch application as a server host (shorthand)")
	flag.BoolVar(&setUsrConfArg, "user-config", false, "Ask user to set config on launch")

	flag.Parse()

	log_path := os.Getenv("SRV_LOG_OUTPUT")
	if log_path == "" {
		log.Fatalf("Could not set log output. Please ensure .env file has been setup.")
	}

	fileName := fmt.Sprintf("%v_server.log", time.Now().UTC().Format("2006-01-02"))
	f, err := os.Create(filepath.Join(log_path, fileName))

	if err != nil {
		log.Fatal("Could not create file for log")
	}
	defer f.Close()

	logger = log.New(f, "", log.Lshortfile|log.LstdFlags)

	// try to load conf from a file. if not existing, ask user for details
	conf_path := os.Getenv("USR_CONFIG_PATH")
	if conf_path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			logger.Fatalf("cannot set default user config path: %v", err)
		}
		conf_path = path.Join(home, ".simple_server_user_config")
	}

	cfg := client.SetupClientConfig(conf_path, setUsrConfArg)
	cfg.Logger = logger
	cli := client.NewClient(cfg)

	if hostModeArg {
		var port string

		historySize, err := strconv.ParseUint(os.Getenv("SRV_MSG_HISTORY_SIZE"), 10, 64)
		if err != nil {
			logger.Fatalf("could not parse History size to uint: %v", err)
		}

		maxConnectionLimit, err := strconv.ParseUint(os.Getenv("SRV_MAX_CONNECTIONS"), 10, 64)
		if err != nil {
			logger.Fatalf("could not parse Max connection limit to uint: %v", err)
		}

		if portArg == 0 {
			port = os.Getenv("SRV_PORT")
			valid, msg := validatePortString(port)
			if msg != "" {
				logger.Println(msg)
			}
			if !valid {
				os.Exit(1)
			}
		} else {
			valid, msg := validatePort(portArg)
			if msg != "" {
				logger.Println(msg)
			}
			if !valid {
				os.Exit(1)
			}
			port = fmt.Sprintf("%d", portArg)
		}

		srv, err := server.NewServer(port, uint(historySize), logger)
		srv.MaxConnectionLimit = uint(maxConnectionLimit)
		if err != nil {
			logger.Fatalln(err)
		}

		go srv.StartListening()
		cli.Connect(fmt.Sprintf("127.0.0.1:%s", port))
		cli.SetAsHost(&srv)
	}
	client.StartTUI(&cli)
}
