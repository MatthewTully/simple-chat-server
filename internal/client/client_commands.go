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
	srvAddr := c.userCmdArg
	c.chatView.Write([]byte(fmt.Sprintf("Attempting to connect to %v\n", srvAddr)))
	c.Connect(srvAddr)
}

func disconnectFromServer(c *Client) {
	conn := c.ActiveConn
	if conn == nil {
		c.cfg.Logger.Println("No active connections")
		c.chatView.Write([]byte("No active connections"))
		return
	}
	c.chatView.Write([]byte(fmt.Sprintf("Disconnecting from %v\n", c.ActiveConn.RemoteAddr().String())))
	c.ActiveConn.Close()
	c.chatView.Write([]byte("Successfully disconnected."))
}

func exitApplication(c *Client) {
	c.cfg.Logger.Println("Closing any active connections..")
	c.chatView.Write([]byte("Closing any active connections.."))
	disconnectFromServer(c)
	c.cfg.Logger.Println("Closing application")
	c.chatView.Write([]byte("Closing application"))
	c.TUI.Stop()
	os.Exit(0)
}

func listUserCommands(c *Client) {
	c.tuiPages.ShowPage("user-commands")
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
		c.userCmdArg = strings.Join(inputArgs[1:], " ")
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
