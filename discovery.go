package main

import (
	"encoding/json"
	"log"
	"net"
	"strconv"
	"time"
)

const (
	discoveryGroup   = "239.255.0.1:9000"
	announceInterval = 3 * time.Second
)

type discoveryMessage struct {
	ID      string `json:"id"`
	Nick    string `json:"nick"`
	TCPPort int    `json:"tcp_port"`
}

func (n *p2pNode) startDiscovery() error {
	groupAddr, err := net.ResolveUDPAddr("udp4", discoveryGroup)
	if err != nil {
		return err
	}

	listenConn, err := net.ListenMulticastUDP("udp4", nil, groupAddr)
	if err != nil {
		return err
	}

	sendConn, err := net.DialUDP("udp4", nil, groupAddr)
	if err != nil {
		listenConn.Close()
		return err
	}

	go n.listenForAnnouncements(listenConn)
	go n.announce(sendConn)

	return nil
}

func (n *p2pNode) announce(conn *net.UDPConn) {
	defer conn.Close()

	message := discoveryMessage{
		ID:      n.id,
		Nick:    n.nick,
		TCPPort: n.port,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to encode discovery message: %v", err)
		return
	}

	ticker := time.NewTicker(announceInterval)
	defer ticker.Stop()

	for {
		_, err := conn.Write(data)
		if err != nil {
			log.Printf("Failed to announce presence: %v", err)
		}

		<-ticker.C
	}
}

func (n *p2pNode) listenForAnnouncements(conn *net.UDPConn) {
	defer conn.Close()

	buffer := make([]byte, 1024)

	for {
		size, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Discovery read error: %v", err)
			return
		}

		var message discoveryMessage

		if err := json.Unmarshal(buffer[:size], &message); err != nil {
			continue
		}

		if message.ID == n.id || message.TCPPort < 1 || message.TCPPort > 65535 {
			continue
		}

		// Only one node initiates the connection.
		if n.id > message.ID {
			continue
		}

		n.mu.Lock()

		_, connected := n.peers[message.ID]
		lastAttempt, attempted := n.lastAttempt[message.ID]

		shouldConnect := !connected &&
			(!attempted || time.Since(lastAttempt) >= 10*time.Second)

		if shouldConnect {
			n.lastAttempt[message.ID] = time.Now()
		}

		n.mu.Unlock()

		if !shouldConnect {
			continue
		}

		address := net.JoinHostPort(
			remoteAddr.IP.String(),
			strconv.Itoa(message.TCPPort),
		)

		log.Printf("Discovered peer: %s at %s", message.Nick, address)

		go n.connectToPeer(address)
	}
}
