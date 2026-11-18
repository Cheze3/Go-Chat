# Go TCP Chat

A terminal chat built from scratch with Go's networking packages. It has two modes:

- **Server mode:** a central TCP server with chat rooms, nicknames and broadcasting. Clients connect with any TCP client such as Netcat.
- **P2P mode:** nodes on the same local network find each other automatically (UDP multicast), connect directly over TCP, and broadcast messages to every connected peer. No server is needed.

The project explores TCP and UDP communication, goroutines, channels, concurrency, and client-server and peer-to-peer architecture in Go.

## Features

**Server mode**

- Accepts multiple simultaneous client connections.
- Multiple chat rooms: clients can create and join different rooms.
- Nicknames, room listing, and broadcasting to the other members of a room.
- Disconnections are cleaned up whether the client quits or the connection drops.
- Input lines are limited to 4096 bytes, and a client that stops reading is disconnected after a 5 second write timeout instead of blocking the server.

**P2P mode**

- Automatic peer discovery with UDP multicast announcements every 3 seconds.
- The TCP connection acts as the acknowledgement: the receiver of an announcement dials the sender's TCP port using the sender's IP address.
- Only one side dials: the node with the lower ID. If two nodes connect to each other at the same time (for example with `/connect`), both keep the connection opened by the lower ID.
- A handshake exchanges node ID and nickname, with a 5 second deadline.
- Disconnected peers are removed and rediscovered automatically.
- Typed lines are broadcast to all connected peers.

## Project Structure

```text
.
├── main.go       # Entry point; selects server or p2p mode
├── server.go     # Server initialization and command handling
├── client.go     # Server-mode client connections and input processing
├── room.go       # Room management and message broadcasting
├── command.go    # Command definitions and identifiers
├── p2p.go        # P2P node: listener, handshake, peers, broadcast, input
├── peer.go       # P2P peer connection and message framing
├── discovery.go  # UDP multicast announcements and peer discovery
└── README.md
```

### Architecture

**Server mode**

1. **Server (`server.go`):** keeps the map of rooms and processes commands from a channel in a single goroutine, so room and client state needs no locks.
2. **Client (`client.go`):** reads lines from one TCP connection, parses commands, and sends them to the server through the channel.
3. **Room (`room.go`):** holds the members of a room and broadcasts messages to everyone except the sender.
4. **Command (`command.go`):** command identifiers and the structure passed from clients to the server.

**P2P mode**

1. **Node (`p2p.go`):** listens on a TCP port, runs the handshake, registers peers, and runs one read loop per connection. Duplicate connections to the same node are resolved by the lower-ID rule.
2. **Peer (`peer.go`):** one connection, with newline-delimited JSON messages and a mutex that serializes writes.
3. **Discovery (`discovery.go`):** joins multicast group `239.255.0.1:9000`, announces `{id, nick, tcp_port}` every 3 seconds, and dials newly discovered peers with a lower ID than its own.

## Getting Started

### Prerequisites

- [Go](https://go.dev/dl/) installed.
- A terminal application.
- For server mode, a TCP client such as Netcat (`nc`).

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

### Running in Server Mode

Start the server:

```bash
go run . server
```

The server listens for TCP connections on port `8080`.

Expected output:

```text
Server started on :8080
```

In another terminal, connect with Netcat (open several terminals to simulate several clients):

```bash
nc localhost 8080
```

### Running in P2P Mode

Start a node:

```bash
go run . p2p -nick <name> -port <port>
```

`-nick` defaults to `Anonymous` and `-port` (the TCP listening port) defaults to `8081`.

To try it on one machine, start two nodes with different ports:

```bash
go run . p2p -nick alice -port 8081
go run . p2p -nick bob -port 8082
```

Within about 10 seconds each node should log `Discovered peer` and `Connected to peer`. Then type a line in either terminal to send it to the other.

To connect manually (also useful to check whether a problem is discovery or TCP):

```text
/connect 192.168.1.20:8082
```

#### P2P commands

| Input | Description |
|---|---|
| any text | Broadcast the line to all connected peers |
| `/connect <host:port>` | Connect to a peer manually |
| `/peers` | List connected peers |

#### Network requirements

- Devices must be on the same subnet. Multicast is sent with TTL 1 and does not cross routers.
- The router must allow device-to-device traffic and multicast. Client isolation and some guest or mesh networks block both.
- Firewalls must allow inbound UDP port `9000` and the chosen TCP port.
- On machines with several network interfaces (VPN, Docker, WSL), the operating system picks the multicast interface, which may not be the Wi-Fi one.

## Server Commands

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

- **TCP networking (`net`):** client-server and peer-to-peer connections.
- **UDP multicast:** announcing presence to every node on the local network.
- **Goroutines:** one per connection, plus discovery loops.
- **Channels:** passing commands from client goroutines to the server loop.
- **Structs and methods:** organizing server, client, room, command and peer data.
- **Maps:** tracking rooms, room members and connected peers.
- **Interfaces:** Go's networking interfaces such as `net.Conn`.
- **Synchronization (`sync.Mutex`, `sync.RWMutex`):** serializing writes to a connection and guarding the peer map.
- **Deadlines:** handshake, write and read limits so a stalled connection cannot block the program.

## Future Improvements

- Private messaging between individual clients or peers.
- Room passwords and access control.
- Client authentication.
- Improved command parsing and validation, and nickname uniqueness.
- Choosing the multicast interface explicitly, or using every active interface.
- A stop mechanism for discovery, and pruning of the discovery retry table.
- Message timestamps.
- A graphical client interface.
- Automated tests for room management, broadcasting and peer connection handling.

## License

No license yet