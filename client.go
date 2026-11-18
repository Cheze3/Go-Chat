package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	maxLineLen         = 4096
	clientWriteTimeout = 5 * time.Second
)

type client struct {
	conn    net.Conn
	nick    string
	room    *room
	commads chan<- command
	mu      sync.Mutex

	closed bool
}

func (c *client) readInput() {
	scanner := bufio.NewScanner(c.conn)
	scanner.Buffer(make([]byte, 0, 1024), maxLineLen)

	for scanner.Scan() {
		args := strings.Fields(scanner.Text())
		if len(args) == 0 {
			continue
		}

		cmd := args[0]

		switch cmd {
		case "/nick":
			c.commads <- command{id: CMD_NICK, client: c, args: args}
		case "/join":
			c.commads <- command{id: CMD_JOIN, client: c, args: args}
		case "/rooms":
			c.commads <- command{id: CMD_ROOMS, client: c, args: args}
		case "/msg":
			c.commads <- command{id: CMD_MSG, client: c, args: args}
		case "/quit":
			c.commads <- command{id: CMD_QUIT, client: c, args: args}
		default:
			c.err(fmt.Errorf("Unknown command: %s", cmd))
		}
	}

	if err := scanner.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			c.err(fmt.Errorf("Line too long (max %d bytes)", maxLineLen))
		}

		if !errors.Is(err, net.ErrClosed) {
			log.Printf("Read error from %v: %v", c.conn.RemoteAddr(), err)
		}
	}

	// The connection ended. The server ignores this if the client already quit.
	c.commads <- command{id: CMD_QUIT, client: c}
}

func (c *client) msg(msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.conn.SetWriteDeadline(time.Now().Add(clientWriteTimeout))

	if _, err := fmt.Fprintln(c.conn, msg); err != nil {
		c.conn.Close()
	}
}

func (c *client) err(err error) {
	c.msg("Error: " + err.Error())
}
