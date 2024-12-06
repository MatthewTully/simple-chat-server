package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
)

var validColours = []string{"red", "orange", "blue", "green", "yellow", "pink", "purple", "black", "white", "grey"}

func SetupClientConfig(filePath string, manualSet bool) *ClientConfig {
	usr_f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Could not action usr config: %v", err)
	}
	fi, err := usr_f.Stat()
	if err != nil {
		log.Fatalf("Could not action usr config: %v", err)
	}
	usr_f.Close()

	var cfg ClientConfig
	if manualSet || (fi.Size() == 0) {
		cfg.Username, cfg.UserColour = AskUserDetailsCLI()
		jsonOut, err := json.Marshal(cfg)
		if err != nil {
			log.Fatal(err)
		}
		err = os.WriteFile(filePath, jsonOut, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		usr_conf, err := os.ReadFile(filePath)
		if err != nil {
			log.Fatal(err)
		}
		err = json.Unmarshal(usr_conf, &cfg)
		if err != nil {
			log.Fatal(err)
		}
	}
	return &cfg
}

func AskUserDetailsCLI() (string, string) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Configure User details...")
	fmt.Printf("Please enter a user name (Max 32 char): ")
	scanner.Scan()
	username := scanner.Text()
	retry := true
	for retry {
		if len(username) > 32 {
			fmt.Printf("Username too long, please try again: ")
			scanner.Scan()
			username = scanner.Text()
		} else {
			fmt.Printf("Are you happy with this username (%v)? Y/N\n", username)
			scanner.Scan()
			confirm := scanner.Text()
			switch strings.ToLower(confirm) {
			case "y":
				retry = false
			case "n":
				fmt.Printf("Please enter a user name (Max 32 char): ")
				scanner.Scan()
				username = scanner.Text()
			default:
				fmt.Println("Invalid response, please use Y or N")
			}
		}
	}

	fmt.Printf("Valid Colours:\n")
	for _, colour := range validColours {
		fmt.Printf("  - %v\n", colour)
	}
	fmt.Printf("Please enter a colour to represent your username:\n")
	scanner.Scan()
	colour := scanner.Text()
	retry = true
	for retry {
		if !slices.Contains(validColours, colour) {
			fmt.Printf("Unsupported Colour, please try again: ")
			scanner.Scan()
			colour = scanner.Text()
		} else {
			fmt.Printf("Are you happy with this Colour (%v)? Y/N\n", colour)
			scanner.Scan()
			confirm := scanner.Text()
			switch strings.ToLower(confirm) {
			case "y":
				retry = false
			case "n":
				fmt.Printf("Please enter a colour to represent your username: ")
				scanner.Scan()
				colour = scanner.Text()
			default:
				fmt.Println("Invalid response, please use Y or N")
			}
		}
	}
	return username, colour
}
