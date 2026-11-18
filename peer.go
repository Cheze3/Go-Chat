package main

import (
	"bufio"
	"encoding/json"
	"net"
	"sync"
	"time"
)

const writeTimeout = 5 * time.Second

// chatMessage is the only message type sent after the handshake.
// The sender's nickname is not part of it: the receiver uses the nickname
// from the handshake, so a peer cannot impersonate another one per message.
type chatMessage struct {
	Text string `json:"text"`
}

type peer struct {
	id   string
	nick string
	conn net.Conn

	scanner   *bufio.Scanner
	initiator string

	mu sync.Mutex
}

func writeJSON(conn net.Conn, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	conn.SetWriteDeadline(time.Now().Add(writeTimeout))

	_, err = conn.Write(append(data, '\n'))
	return err
}

func (p *peer) send(msg chatMessage) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	return writeJSON(p.conn, msg)
}
