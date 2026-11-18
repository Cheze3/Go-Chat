# Go TCP Chat Room

A simple multi-client chat room built from scratch using Go's TCP networking capabilities. The server allows multiple clients to connect, choose nicknames, join chat rooms, exchange messages, and disconnect.

The project explores TCP communication, goroutines, channels, concurrency, and client-server architecture in Go.

## Features

- **TCP Server:** Accepts multiple simultaneous client connections.
- **Multiple Chat Rooms:** Clients can create and join different chat rooms.
- **Nicknames:** Clients can set custom usernames.
- **Room Broadcasting:** Messages are broadcast to all other members in the same room.
- **Room Listing:** View all available chat rooms.
- **Graceful Disconnection:** Clients can leave rooms and disconnect from the server.
- **Concurrent Connections:** Uses goroutines to handle multiple clients simultaneously.

## Project Structure

```text
.
├── main.go       # Entry point; starts the TCP server
├── server.go     # Server initialization and command handling
├── client.go     # Client connections and input processing
├── room.go       # Room management and message broadcasting
├── command.go    # Command definitions and identifiers
└── README.md
```

### Architecture

The program is divided into four main components:

**1. Server (`server.go`)**
- Maintains a map of available chat rooms.
- Manages client connections.
- Processes commands received through a channel.
- Handles nickname changes, room management, and message broadcasting.

**2. Client (`client.go`)**
- Represents an individual TCP client.
- Reads incoming messages from the TCP connection.
- Parses user commands.
- Sends commands to the server through a channel.
- Handles outgoing messages from the server.

**3. Room (`room.go`)**
- Maintains a collection of connected clients.
- Broadcasts messages to other members in the room.
- Keeps communication isolated between different rooms.

**4. Command (`command.go`)**
- Defines command identifiers.
- Provides a structure for passing client commands to the server.

## Getting Started

### Prerequisites

- [Go](https://go.dev/dl/) installed.
- A terminal application.
- A TCP client such as Netcat (`nc`).

### Installation

Clone the repository:

```bash
git clone <repository-url>
cd <project-directory>
```

Initialize the Go module if one does not already exist:

```bash
go mod init tcp-chat-room
```

### Running the Server

Start the server:

```bash
go run .
```

The server listens for TCP connections on port `8080`.

Expected output:

```text
Server started on :8080
```

### Connecting to the Server

Open another terminal and connect using Netcat:

```bash
nc localhost 8080
```

You can open multiple terminal windows to simulate multiple clients.

## Available Commands

| Command | Description | Example |
|---|---|---|
| `/nick <name>` | Set your nickname | `/nick Binh` |
| `/join <room>` | Join or create a chat room | `/join general` |
| `/rooms` | List available chat rooms | `/rooms` |
| `/msg <message>` | Send a message to your current room | `/msg Hello everyone!` |
| `/quit` | Disconnect from the server | `/quit` |

### Example Session

**Client 1:**

```text
/nick Alice
Your nickname has been set to Alice

/join general
You have joined general
```

**Client 2:**

```text
/nick Bob
Your nickname has been set to Bob

/join general
You have joined general
Alice has joined the room
```

**Client 1:**

```text
/msg Hello Bob!
```

**Client 2:**

```text
Alice: Hello Bob!
```

## Technical Concepts

This project explores several fundamental Go concepts:

- **TCP Networking (`net`):** Establishing server-client connections using TCP.
- **Goroutines:** Handling multiple client connections concurrently.
- **Channels:** Passing commands between client goroutines and the server.
- **Structs and Methods:** Organizing server, client, room, and command data.
- **Maps:** Tracking available rooms and their members.
- **Interfaces:** Using Go's networking interfaces such as `net.Conn`.
- **Synchronization (`sync.Mutex`):** Preventing concurrent writes to client connections.

## Future Improvements

Potential extensions to the project:

- Private messaging between individual clients.
- Room passwords and access control.
- Client authentication.
- Improved command parsing and validation.
- Better handling of unexpected client disconnections.
- Empty room cleanup.
- Message timestamps.
- A graphical client interface.
- Automated tests for room management and message broadcasting.

## License

This project is open source and available under the MIT License.