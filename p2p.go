package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	handshakeTimeout = 5 * time.Second
	dialTimeout      = 5 * time.Second
	maxMessageSize   = 64 * 1024
	maxTextLen       = 4096
	maxNickLen       = 32
	maxIDLen         = 64
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
	log.Printf("Type a message to broadcast it. Commands: /connect <host:port>, /peers")

	// Start reading input only once the node is ready.
	go n.readInput()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}

			log.Printf("Failed to accept connection: %v", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		go n.handleConnection(conn, false)
	}
}

func (n *p2pNode) handleConnection(conn net.Conn, outbound bool) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 4096), maxMessageSize)

	remote, err := n.exchangeHandshake(conn, scanner)
	if err != nil {
		log.Printf("Handshake with %s failed: %v", conn.RemoteAddr(), err)
		return
	}

	if remote.ID == n.id {
		log.Printf("Rejected connection to self")
		return
	}

	initiator := remote.ID
	if outbound {
		initiator = n.id
	}

	p := &peer{
		id:        remote.ID,
		nick:      remote.Nick,
		conn:      conn,
		scanner:   scanner,
		initiator: initiator,
	}

	if !n.addPeer(p) {
		return
	}

	log.Printf("Connected to peer: %s (%s)", p.nick, p.id)

	n.readLoop(p)
	n.removePeer(p)

	log.Printf("Disconnected from peer: %s (%s)", p.nick, p.id)
}

// exchangeHandshake sends our identity and reads the remote one, with a
// deadline so a silent connection cannot hang forever.
func (n *p2pNode) exchangeHandshake(conn net.Conn, scanner *bufio.Scanner) (handshake, error) {
	var remote handshake

	conn.SetDeadline(time.Now().Add(handshakeTimeout))
	defer conn.SetDeadline(time.Time{})

	err := writeJSON(conn, handshake{ID: n.id, Nick: n.nick})
	if err != nil {
		return remote, fmt.Errorf("send: %w", err)
	}

	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return remote, fmt.Errorf("receive: %w", err)
		}
		return remote, fmt.Errorf("receive: %w", io.EOF)
	}

	if err := json.Unmarshal(scanner.Bytes(), &remote); err != nil {
		return remote, fmt.Errorf("decode: %w", err)
	}

	if remote.ID == "" || len(remote.ID) > maxIDLen {
		return remote, errors.New("invalid node ID")
	}

	remote.Nick = sanitize(remote.Nick, maxNickLen)
	if remote.Nick == "" {
		remote.Nick = "Anonymous"
	}

	return remote, nil
}

// addPeer registers p. If a connection to the same node already exists, the
// one opened by the node with the lower ID wins. Both ends apply the same
// rule, so after a simultaneous dial they keep the same connection.
// It returns false if p lost and must be closed by the caller.
func (n *p2pNode) addPeer(p *peer) bool {
	n.mu.Lock()

	existing, exists := n.peers[p.id]
	if exists && existing.initiator <= p.initiator {
		n.mu.Unlock()
		return false
	}

	n.peers[p.id] = p
	n.mu.Unlock()

	if exists {
		// Its read loop ends; removePeer ignores it because it is no longer registered.
		existing.conn.Close()
	}

	return true
}

// removePeer deletes p only if it is still the registered connection.
func (n *p2pNode) removePeer(p *peer) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.peers[p.id] == p {
		delete(n.peers, p.id)
	}
}

func (n *p2pNode) readLoop(p *peer) {
	for p.scanner.Scan() {
		var msg chatMessage

		if err := json.Unmarshal(p.scanner.Bytes(), &msg); err != nil {
			log.Printf("Invalid message from %s: %v", p.nick, err)
			continue
		}

		fmt.Printf("[%s] %s\n", p.nick, sanitize(msg.Text, maxTextLen))
	}

	if err := p.scanner.Err(); err != nil && !errors.Is(err, net.ErrClosed) {
		log.Printf("Read error from %s: %v", p.nick, err)
	}
}

func (n *p2pNode) broadcast(text string) {
	if len(text) > maxTextLen {
		fmt.Printf("Message too long (max %d bytes)\n", maxTextLen)
		return
	}

	n.mu.RLock()
	peers := make([]*peer, 0, len(n.peers))
	for _, p := range n.peers {
		peers = append(peers, p)
	}
	n.mu.RUnlock()

	if len(peers) == 0 {
		fmt.Println("No peers connected")
		return
	}

	// Send concurrently so one stalled peer does not delay the others.
	var wg sync.WaitGroup

	for _, p := range peers {
		wg.Add(1)

		go func(p *peer) {
			defer wg.Done()

			if err := p.send(chatMessage{Text: text}); err != nil {
				log.Printf("Failed to send to %s: %v", p.nick, err)
				p.conn.Close() // the read loop removes the peer
			}
		}(p)
	}

	wg.Wait()
}

func (n *p2pNode) printPeers() {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if len(n.peers) == 0 {
		fmt.Println("No peers connected")
		return
	}

	for _, p := range n.peers {
		fmt.Printf("%s (%s) %s\n", p.nick, p.id, p.conn.RemoteAddr())
	}
}

// connectToPeer dials address and serves the connection until it closes.
// It blocks, so callers run it in a goroutine.
func (n *p2pNode) connectToPeer(address string) {
	conn, err := net.DialTimeout("tcp", address, dialTimeout)
	if err != nil {
		log.Printf("Failed to connect to %s: %v", address, err)
		return
	}

	n.handleConnection(conn, true)
}

func (n *p2pNode) readInput() {
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())

		switch {
		case input == "":
		case strings.HasPrefix(input, "/connect "):
			address := strings.TrimSpace(strings.TrimPrefix(input, "/connect "))
			go n.connectToPeer(address)
		case input == "/peers":
			n.printPeers()
		case strings.HasPrefix(input, "/"):
			fmt.Println("Unknown command. Available: /connect <host:port>, /peers")
		default:
			n.broadcast(input)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Failed to read input: %v", err)
	}
}

// sanitize removes control characters (including terminal escape sequences)
// and truncates to maxRunes runes.
func sanitize(s string, maxRunes int) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)

	runes := []rune(s)
	if len(runes) > maxRunes {
		runes = runes[:maxRunes]
	}

	return string(runes)
}
