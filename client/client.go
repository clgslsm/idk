package client

import (
	"bufio"
	"fmt"
	"net"
	"time"
)

func StartDownload(torrentFile string) {
	fmt.Println("Starting download for:", torrentFile)

	// Parse torrent file
	pieces, err := parseTorrentFile(torrentFile)
	if err != nil {
		fmt.Printf("Failed to parse torrent file: %v\n", err)
		return
	}

	// Simulate downloading pieces from peers
	for _, piece := range pieces {
		fmt.Printf("Downloading piece: %s...\n", piece)

		// Example: Connect to a peer
		err := requestPieceFromPeer("localhost:8080", piece)
		if err != nil {
			fmt.Printf("Error downloading piece %s: %v\n", piece, err)
		} else {
			fmt.Printf("Successfully downloaded piece: %s\n", piece)
		}

		time.Sleep(1 * time.Second) // Simulated delay
	}

	fmt.Println("Download complete!")
}

func parseTorrentFile(filePath string) ([]string, error) {
	// Simulated pieces
	return []string{"piece1", "piece2", "piece3"}, nil
}

func requestPieceFromPeer(address, piece string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("error connecting to peer: %v", err)
	}
	defer conn.Close()

	// Send request for the piece
	message := fmt.Sprintf("Requesting piece: %s\n", piece)
	_, err = conn.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}

	// Read response from the peer
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading response: %v", err)
	}

	fmt.Printf("Response from peer: %s", response)
	return nil
}

func TestConnection(address string) error {
	// Set timeout for the entire operation
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return fmt.Errorf("connection failed: %v", err)
	}
	defer conn.Close()

	// Set read/write deadlines
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Send a test message
	_, err = conn.Write([]byte("test\n")) // Add newline as message delimiter
	if err != nil {
		return fmt.Errorf("failed to send test message: %v", err)
	}

	// Read response
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	fmt.Printf("Received response: %s", response)
	return nil
}
