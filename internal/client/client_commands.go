package client

import (
	"fmt"
	"os"
	"strings"
)

type userCommand struct {
	name        string
	description string
	callback    func(*Client)
}

func getUserCommands() map[string]userCommand {
	return map[string]userCommand{
		"\\connect": {
			name:        "\\connect",
			description: "connect to a server",
			callback:    connectToServer,
		},
		"\\disconnect": {
			name:        "\\disconnect",
			description: "disconnect from a server",
			callback:    disconnectFromServer,
		},
		"\\exit": {
			name:        "\\exit",
			description: "close the application",
			callback:    exitApplication,
		},
		"\\list-user-commands": {
			name:        "\\list-user-commands",
			description: "List available commands",
			callback:    listUserCommands,
		},
	}
}

func connectToServer(c *Client) {
	srvAddr := c.LastCommand
	fmt.Printf("Attempting to connect to %v\n", srvAddr)
	c.Connect(srvAddr)
}

func disconnectFromServer(c *Client) {
	conn := c.ActiveConn
	if conn == nil {
		fmt.Println("No active connections")
		return
	}
	fmt.Printf("Disconnecting from %v\n", c.ActiveConn.RemoteAddr().String())
	c.ActiveConn.Close()
	fmt.Println("Successfully disconnected.")
}

func exitApplication(c *Client) {
	fmt.Println("Closing any active connections..")
	disconnectFromServer(c)
	fmt.Println("Closing application")
	os.Exit(0)
}

func listUserCommands(c *Client) {
	usrCmdMap := getUserCommands()
	fmt.Println("\n#------------------------------------------------#")
	fmt.Printf("Available User commands:\n\n")
	for _, cmd := range usrCmdMap {
		fmt.Printf("  %s - %s\n", cmd.name, cmd.description)
	}
	fmt.Println("#------------------------------------------------#")
}

func actionInput(c *Client, usrInput string) {
	usrCmdMap := getUserCommands()
	inputArgs := strings.Fields((usrInput))
	if len(inputArgs) == 0 {
		return
	}
	cmd := inputArgs[0]
	if strings.HasPrefix(cmd, "\\") {
		clientCmd, exists := usrCmdMap[cmd]
		if !exists {
			//fmt.Printf("\n%s is not a valid user command. Use \\list-user-commands to see available user commands.", cmd)
			return
		}
		c.LastCommand = strings.Join(inputArgs[1:], " ")
		clientCmd.callback(c)
		return
	}
	err := c.SendMessageToServer([]byte(usrInput))
	if err != nil {
		//TODO show error sending, ask to try again.
		return
	}
	c.PushMessageToChatView(usrInput)
}
