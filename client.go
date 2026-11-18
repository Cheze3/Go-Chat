package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
)

type client struct {
	conn    net.Conn
	nick    string
	room    *room
	commads chan<- command
	mu      sync.Mutex
}

func (c *client) readInput() {
	reader := bufio.NewReader(c.conn)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Client disconnected: %v", c.conn.RemoteAddr())

			c.commads <- command{
				id:     CMD_QUIT,
				client: c,
			}
			return
		}

		msg = strings.Trim(msg, "\r\n")

		args := strings.Fields(msg)
		cmd := strings.TrimSpace(args[0])

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
}

func (c *client) msg(msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Fprintln(c.conn, msg+"\n")
}

func (c *client) err(err error) {
	c.msg("Error: " + err.Error())
}
