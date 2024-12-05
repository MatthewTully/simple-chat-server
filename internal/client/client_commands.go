package client

import (
	"fmt"
	"maps"
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
			description: "Connect to a server",
			callback:    connectToServer,
		},
		"\\disconnect": {
			name:        "\\disconnect",
			description: "Disconnect from a server",
			callback:    disconnectFromServer,
		},
		"\\exit": {
			name:        "\\exit",
			description: "Close the application",
			callback:    exitApplication,
		},
		"\\list-user-commands": {
			name:        "\\list-user-commands",
			description: "List available commands",
			callback:    listUserCommands,
		},
		"\\whisper": {
			name:        "\\whisper",
			description: "Send a message directly to a user",
			callback:    whisperMsgToUser,
		},
	}
}

func getHostCommands() map[string]userCommand {
	return map[string]userCommand{
		"\\kick": {
			name:        "\\kick",
			description: "Disconnect the specified user",
			callback:    kickUser,
		},
		"\\ban": {
			name:        "\\ban",
			description: "Disconnect user and add their IP to the blacklist",
			callback:    banUser,
		},
	}
}

func kickUser(c *Client) {
	if !c.Host {
		return
	}
	usr := c.userCmdArg
	c.HostServer.CloseConnectionForUser(usr)
}

func banUser(c *Client) {
	if !c.Host {
		return
	}
	usr := c.userCmdArg
	c.HostServer.BanUser(usr)
}

func connectToServer(c *Client) {
	srvAddr := c.userCmdArg
	c.PushToChatView(fmt.Sprintf("Attempting to connect to %v\n", srvAddr))
	c.Connect(srvAddr)
}

func disconnectFromServer(c *Client) {
	conn := c.ActiveConn
	if conn == nil {
		c.PushToChatView("No active connections")
		return
	}
	c.PushToChatView(fmt.Sprintf("Disconnecting from %v\n", c.ActiveConn.RemoteAddr().String()))
	c.SendDisconnectionRequest()
	c.ActiveConn.Close()
	c.PushToChatView("Successfully disconnected.")
}

func exitApplication(c *Client) {
	c.PushToChatView("Closing any active connections..")
	disconnectFromServer(c)
	c.PushToChatView("Closing application")
	c.TUI.Stop()
	os.Exit(0)
}

func listUserCommands(c *Client) {
	if c.Host {
		c.tuiPages.ShowPage("host-user-commands")
		return
	}
	c.tuiPages.ShowPage("user-commands")
}

func whisperMsgToUser(c *Client) {
	msg := c.userCmdArg
	if len(msg) > 0 {
		msg := msg + "\n"
		c.SendWhisperToServer([]byte(msg))
		c.PushSentMessageToChatView(msg)
	}
}

func actionInput(c *Client, usrInput string) {
	usrCmdMap := getUserCommands()
	if c.Host {
		maps.Copy(usrCmdMap, getHostCommands())
	}
	inputArgs := strings.Fields((usrInput))
	if len(inputArgs) == 0 {
		return
	}
	cmd := inputArgs[0]
	if strings.HasPrefix(cmd, "\\") {
		clientCmd, exists := usrCmdMap[cmd]
		if !exists {
			c.PushToChatView(fmt.Sprintf("%s is not a valid user command. Use \\list-user-commands to see available user commands.", cmd))
			return
		}
		c.userCmdArg = strings.Join(inputArgs[1:], " ")
		clientCmd.callback(c)
		return
	}
	err := c.SendMessageToServer([]byte(usrInput))
	if err != nil {
		c.PushToChatView("Could not send message. Please try again.")
		return
	}
	c.PushSentMessageToChatView(usrInput)
}
