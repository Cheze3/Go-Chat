package main

import (
	"net"
	"sync"
)

type peer struct {
	id   string
	nick string
	conn net.Conn

	mu sync.Mutex
}
