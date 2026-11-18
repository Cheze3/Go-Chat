package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

type handshake struct {
	ID   string `json:"id"`
	Nick string `json:"nick"`
}

type p2pNode struct {
	id   string
	nick string
	port int

	peers       map[string]*peer
	mu          sync.RWMutex
	lastAttempt map[string]time.Time
}

func newP2PNode(nick string, port int) (*p2pNode, error) {
	idBytes := make([]byte, 8)

	_, err := rand.Read(idBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate node ID: %w", err)
	}

	return &p2pNode{
		id:          hex.EncodeToString(idBytes),
		nick:        nick,
		port:        port,
		peers:       make(map[string]*peer),
		lastAttempt: make(map[string]time.Time),
	}, nil
}

func (n *p2pNode) start() {
	go n.readInput()
	addr := fmt.Sprintf(":%d", n.port)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to start P2P listener: %v", err)
	}
	defer listener.Close()

	if err := n.startDiscovery(); err != nil {
		log.Printf("Failed to start discovery: %v", err)
	} else {
		log.Println("Peer discovery started")
	}

	log.Printf("P2P node started")
	log.Printf("Nickname: %s", n.nick)
	log.Printf("Node ID: %s", n.id)
	log.Printf("Listening on TCP port: %d", n.port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		go n.handleConnection(conn)
	}
}

func (n *p2pNode) handleConnection(conn net.Conn) {
	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	// Send our identity.
	err := encoder.Encode(handshake{
		ID:   n.id,
		Nick: n.nick,
	})
	if err != nil {
		log.Printf("Failed to send handshake: %v", err)
		conn.Close()
		return
	}

	// Receive the remote node's identity.
	var remote handshake

	err = decoder.Decode(&remote)
	if err != nil {
		log.Printf("Failed to receive handshake: %v", err)
		conn.Close()
		return
	}

	if remote.ID == n.id {
		log.Printf("Rejected connection to self")
		conn.Close()
		return
	}

	n.mu.Lock()

	// Prevent registering the same node twice.
	if _, exists := n.peers[remote.ID]; exists {
		n.mu.Unlock()
		conn.Close()
		return
	}

	n.peers[remote.ID] = &peer{
		id:   remote.ID,
		nick: remote.Nick,
		conn: conn,
	}

	n.mu.Unlock()

	log.Printf("Connected to peer: %s (%s)", remote.Nick, remote.ID)
}

func (n *p2pNode) connectToPeer(address string) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Printf("Failed to connect to %s: %v", address, err)
		return
	}

	go n.handleConnection(conn)
}

func (n *p2pNode) readInput() {
	reader := bufio.NewScanner(os.Stdin)

	for reader.Scan() {
		input := strings.TrimSpace(reader.Text())

		if strings.HasPrefix(input, "/connect ") {
			address := strings.TrimPrefix(input, "/connect ")
			n.connectToPeer(address)
		}
	}

	if err := reader.Err(); err != nil {
		log.Printf("Failed to read input: %v", err)
	}
}
