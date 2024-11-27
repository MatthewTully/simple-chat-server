package client

import (
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
	c.cfg.Logger.Printf("Attempting to connect to %v\n", srvAddr)
	c.Connect(srvAddr)
}

func disconnectFromServer(c *Client) {
	conn := c.ActiveConn
	if conn == nil {
		c.cfg.Logger.Println("No active connections")
		return
	}
	c.cfg.Logger.Printf("Disconnecting from %v\n", c.ActiveConn.RemoteAddr().String())
	c.ActiveConn.Close()
	c.cfg.Logger.Println("Successfully disconnected.")
}

func exitApplication(c *Client) {
	c.cfg.Logger.Println("Closing any active connections..")
	disconnectFromServer(c)
	c.cfg.Logger.Println("Closing application")
	os.Exit(0)
}

func listUserCommands(c *Client) {
	usrCmdMap := getUserCommands()
	c.cfg.Logger.Println("\n#------------------------------------------------#")
	c.cfg.Logger.Printf("Available User commands:\n\n")
	for _, cmd := range usrCmdMap {
		c.cfg.Logger.Printf("  %s - %s\n", cmd.name, cmd.description)
	}
	c.cfg.Logger.Println("#------------------------------------------------#")
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
			c.cfg.Logger.Printf("\n%s is not a valid user command. Use \\list-user-commands to see available user commands.", cmd)
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
