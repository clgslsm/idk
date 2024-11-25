package server

import (
	"bufio"
	"fmt"
	"net"
)

func StartServer(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("error starting TCP server: %v", err)
	}
	defer listener.Close()

	fmt.Printf("Server listening on %s...\n", address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
			continue
		}

		// Handle each connection in a new goroutine
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// Read data from the peer
	reader := bufio.NewReader(conn)
	message, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Error reading from connection: %v\n", err)
		return
	}

	// Example: Log received message
	fmt.Printf("Received: %s", message)

	// Respond to the peer
	conn.Write([]byte("Message received\n"))
}
