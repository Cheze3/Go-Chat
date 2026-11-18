package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  go run . server")
		fmt.Println("  go run . p2p -nick <name> -port <port>")
		return
	}
	switch os.Args[1] {
	case "server":
		runServer()
	case "p2p":
		runP2PMode(os.Args[2:])
	default:
		fmt.Println("Unknown command:", os.Args[1])
	}
}

func runServer() {
	s := newServer()
	go s.run()

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Failed to start server: %v", err.Error())
	}
	defer listener.Close()
	log.Printf("Server started on :8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err.Error())
			continue
		}

		go s.newClient(conn)
	}
}

func runP2PMode(args []string) {
	fs := flag.NewFlagSet("p2p", flag.ExitOnError)

	nick := fs.String("nick", "Anonymous", "Your nickname")
	port := fs.Int("port", 8081, "TCP listening port")

	fs.Parse(args)

	if *port < 1 || *port > 65535 {
		log.Fatal("Port must be between 1 and 65535")
	}

	node, err := newP2PNode(*nick, *port)
	if err != nil {
		log.Fatalf("Failed to initialize P2P node: %v", err)
	}

	node.start()
}