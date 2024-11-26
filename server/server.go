package server

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

// Simulated data store for the server
var pieces = map[string]string{
	"piece1": "This is data for piece 1.",
	"piece2": "This is data for piece 2.",
	"piece3": "This is data for piece 3.",
}

// StartServer initializes the server to handle peer requests.
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

	// Create a buffered reader to process incoming data
	reader := bufio.NewReader(conn)

	for {
		// Read client request
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading from connection: %v\n", err)
			return
		}
		message = strings.TrimSpace(message)

		// Process the message based on its type
		switch {
		case strings.HasPrefix(message, "HANDSHAKE:"):
			handleHandshake(conn, message)

		case message == "CHECK_PIECES":
			handlePieceAvailability(conn)

		case strings.HasPrefix(message, "Requesting piece:"):
			handlePieceRequest(conn, message)

		default:
			fmt.Printf("Unknown message: %s\n", message)
			conn.Write([]byte("ERROR: Unknown message\n"))
		}
	}
}

func handleHandshake(conn net.Conn, message string) {
	fmt.Printf("Received handshake message: %s\n", message)

	// Respond to the handshake
	conn.Write([]byte("OK\n"))
}

func handlePieceAvailability(conn net.Conn) {
	fmt.Println("Received piece availability request")

	// Simulate the server has all pieces
	conn.Write([]byte("HAVE_PIECES\n"))
}

func handlePieceRequest(conn net.Conn, message string) {
	fmt.Printf("Received piece request: %s\n", message)

	// Extract the requested piece from the message
	pieceKey := strings.TrimPrefix(message, "Requesting piece: ")
	pieceData, found := pieces[pieceKey]
	if !found {
		// If the requested piece is not found, respond with an error
		conn.Write([]byte("ERROR: Piece not found\n"))
		return
	}

	// Send the piece data to the client
	conn.Write([]byte(pieceData + "\n"))
	fmt.Printf("Sent piece data for %s\n", pieceKey)
}
