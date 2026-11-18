package main

import "net"

type room struct {
	name    string
	members map[net.Conn]*client
}

func (r *room) broadcast(sender *client, msg string) {
	for conn, m := range r.members {
		if conn != sender.conn {
			m.msg(msg)
		}
	}
}
