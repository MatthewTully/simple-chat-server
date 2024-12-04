# User Commands

Below is a full list of the currently implemented user commands.

To enter a command, enter `\` followed by the command (no space). 
```
Example: 

\connect 127.0.0.1:8144

```

## Basic commands

Basic commands that can be used anywhere in the application by any user.

```

\connect { server address } - Connect to a server
\disconnect                 - Disconnect from the currently connected server
\exit                       - Close the application. (if the user is connected to a server, it will disconnect first)
\list-user-commands         - List available commands
\user-config                - Change the current user config. The user will need to disconnect and reconnect to the server for the changes to take.

```

## Chat commands

Commands that can be used when connected to a server. 

```
\whisper { username }  - Send a message to the specified user only.

```

## Host commands 

List of commands available to the host of the server.

```

\kick { username }  - Will disconnect the specified user.
\ban { username }   - Disconnect user, and add their IP to the blacklist, preventing them user from reconnecting.
\history            - Prints the entire chat history from the buffer.
\history { number } - Prints n latest messages from the chat history. Where n is the number specified. Must be a uint value.

```
