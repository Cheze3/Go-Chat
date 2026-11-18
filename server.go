package main

import (
	"fmt"
	"log"
	"net"
	"strings"
)

type server struct {
	rooms   map[string]*room
	commads chan command
}

func newServer() *server {
	return &server{
		rooms:   make(map[string]*room),
		commads: make(chan command),
	}
}

func (s *server) run() {
	for cmd := range s.commads {
		if cmd.client.closed {
			continue
		}

		switch cmd.id {
		case CMD_NICK:
			s.nick(cmd.client, cmd.args)
		case CMD_JOIN:
			s.join(cmd.client, cmd.args)
		case CMD_ROOMS:
			s.listRooms(cmd.client)
		case CMD_MSG:
			s.msg(cmd.client, cmd.args)
		case CMD_QUIT:
			s.quit(cmd.client)
		}
	}
}

func (s *server) newClient(conn net.Conn) {
	log.Printf("New client connected: %v", conn.RemoteAddr().String())

	c := &client{
		conn:    conn,
		nick:    "Anonymous",
		room:    nil,
		commads: s.commads,
	}
	c.readInput()
}

func (s *server) nick(c *client, args []string) {
	if len(args) < 2 {
		c.err(fmt.Errorf("Usage: /nick <nickname>"))
		return
	}
	c.nick = args[1]
	c.msg(fmt.Sprintf("Your nickname has been set to %s", c.nick))
}

func (s *server) join(c *client, args []string) {
	if len(args) < 2 {
		c.err(fmt.Errorf("Usage: /join <room>"))
		return
	}

	s.quitRoom(c)

	r, ok := s.rooms[args[1]]
	if !ok {
		r = &room{
			name:    args[1],
			members: make(map[net.Conn]*client),
		}
		s.rooms[args[1]] = r
	}

	r.members[c.conn] = c
	c.room = r

	r.broadcast(c, fmt.Sprintf("%s has joined the room", c.nick))
	c.msg(fmt.Sprintf("You have joined %s", r.name))
}

func (s *server) listRooms(c *client) {
	if len(s.rooms) == 0 {
		c.msg("No rooms available.")
		return
	}

	var roomNames []string
	for name := range s.rooms {
		roomNames = append(roomNames, name)
	}
	c.msg(fmt.Sprintf("Available rooms: %s", strings.Join(roomNames, ", ")))
}

func (s *server) msg(c *client, args []string) {
	if len(args) < 2 {
		c.err(fmt.Errorf("Usage: /msg <message>"))
		return
	}
	if c.room == nil {
		c.err(fmt.Errorf("You are not in a room. Join a room to send messages."))
		return
	}
	c.room.broadcast(c, fmt.Sprintf("%s: %s", c.nick, strings.Join(args[1:], " ")))
}

func (s *server) quit(c *client) {
	c.closed = true

	log.Printf("Client disconnected: %v", c.conn.RemoteAddr().String())
	s.quitRoom(c)
	c.msg("Goodbye!")
	c.conn.Close()
}

func (s *server) quitRoom(c *client) {
	if c.room != nil {
		r := c.room

		delete(r.members, c.conn)
		c.room = nil

		r.broadcast(c, fmt.Sprintf("%s has left the room", c.nick))

		if len(r.members) == 0 {
			delete(s.rooms, r.name)
		}
	}
}
