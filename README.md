# simple-chat-server
A simple chat server built in Go. 

>Server is volatile storage. Message history is lost when pushed from the history buffer, or when the server is shut down. Buffer size is configurable in the settings. When a user joins the server, the history is shared immediately.


## Setup
Create a .env file in the local root and specify the following:
    * SRV_PORT (Port for the server to listen on. Must be valid integer)
    * SRV_MSG_HISTORY_SIZE (Max size of the history buffer. Must be a valid integer)
    * SRV_MAX_CONNECTIONS (Max number of connections to the server. Must be a valid integer)


## Running the server
Run the server with `./simple-chat-server` from the CLI. A port can be specified with the `--port` (or `-p`) flag. This value will override any value provide in the `.env` file for `SRV_PORT`.

## CLI Args
```
    --port int
        Define the port for the server to listen on

    -p int
        Define the port for the server to listen on (shorthand)
```
