# simple-chat-server
A simple chat server built in Go. 

The application can be run as a host server, or a connecting client. Running as a Client is the default mode. 

>Server is volatile storage. Message history is lost when pushed from the history buffer, or when the server is shut down. Buffer size is configurable in the settings. When a user joins the server, the history is shared immediately.


## Setup
Pull the repo using git, or download as a zip, and unzip.

Create a `.env` file in the local root and specify the following:
* SRV_PORT (Port for the server to listen on when hosting. Must be valid integer)
* SRV_MSG_HISTORY_SIZE (Max size of the history buffer. Must be a valid integer)
* SRV_MAX_CONNECTIONS (Max number of connections the server will allow. Must be a valid integer)
* USR_CONFIG_PATH (Where the application will store and retrieve the user preferences config (Username etc.), Default is ~/.simple_server_user_config)

Open a terminal in the directory containing the codebase. Build the application using `go build .`. This will create a simple-chat-server file.

## Running the Client
Client mode is the default state of the application. This can be run with `./simple-chat-server`.

On start up the application will look for the user config file. If it does not exist, it will start first time config and ask for the following:
* Username (Max of 32 Bytes)
* Username Colour (List of valid values will be displayed)

The user config can be manually triggered on startup from the CLI with the flag `-user-config`. Alternatively, it can be configured within the application using the user command `\user-config`

To connect to a server, type `\connect { server connection string }`, where `{ server connection string }` is the address of the server you want to connect to. 

```
Example: 

\connect 127.0.0.1:8144

```

### User commands
To interact with the client, the user can use ***user commands***. To enter a command, enter `\` followed by the command (no space). 

```
Example Commands:

\connect - Connect to a server
\disconnect - Disconnect from a server
\exit - Close the application
\user-config - Change the current user config. The user will need to disconnect and reconnect to the server for the changes to take.

```

A full list of user commands can be found [here](./docs/user_commands.md)

## Running the Server
Run the server with `./simple-chat-server --host` from the CLI to run the application in host mode. This will create a server that others can connect to.
A port can be specified with the `--port` (or `-p`) flag. This value will override any value provide in the `.env` file for `SRV_PORT`. 

Once connected and listening for connections, the application will enter client mode and auto connect to the server. 

> As the host user, the user commands will be expanded to allow administrative control. See [user commands](./docs/user_commands.md) for a full list. 

### CLI Args
```
  --host      Launch application as a server host.
  -h          Launch application as a server host (shorthand).
  
  --port int  Define the port for the server to listen on
  -p     int  Define the port for the server to listen on (shorthand)
```


## Dependences 

Env file loading done by -  github.com/joho/godotenv v1.5.1
Client TUI created using  -  github.com/rivo/tview


