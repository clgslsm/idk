package client

import (
	"bufio"
	"fmt"
	"net"
	"time"

	"tcp-app/torrent"
)

func StartDownload(torrentFile string) {
	fmt.Println("Starting download for:", torrentFile)

	// Parse torrent file using the torrent package
	tf, err := torrent.Open(torrentFile)
	if err != nil {
		fmt.Printf("Error opening torrent file: %v\n", err)
		return
	}

	// Mock the list of peers
	peers := []string{"192.168.0.107:8080"}

	// First, test connection and handshake with peers
	var activePeers []string
	for _, peer := range peers {
		err := TestConnection(peer)
		if err != nil {
			fmt.Printf("Peer %s is not available: %v\n", peer, err)
			continue
		}

		if err := performHandshake(peer, tf.InfoHash[:]); err != nil {
			fmt.Printf("Handshake failed with peer %s: %v\n", peer, err)
			continue
		}
		// Convert tf.PieceHashes from [][20]byte to [][]byte
		var pieceHashes [][]byte
		for _, hash := range tf.PieceHashes {
			pieceHashes = append(pieceHashes, hash[:])
		}

		// Check if peer has the file
		hasPieces, err := checkPieceAvailability(peer, pieceHashes)
		if err != nil {
			fmt.Printf("Failed to check pieces for peer %s: %v\n", peer, err)
			continue
		}

		if hasPieces {
			activePeers = append(activePeers, peer)
		}
	}

	if len(activePeers) == 0 {
		fmt.Println("No available peers found!")
		return
	}

	// Download pieces from active peers
	for i, pieceHash := range tf.PieceHashes {
		fmt.Printf("Downloading piece %d (hash: %x)...\n", i, pieceHash)
		err := requestPieceFromPeer(activePeers[0], fmt.Sprintf("%x", pieceHash))
		if err != nil {
			fmt.Printf("Error downloading piece %d: %v\n", i, err)
		} else {
			fmt.Printf("Successfully downloaded piece %d\n", i)
		}

		time.Sleep(1 * time.Second) // Simulated delay
	}

	fmt.Println("Download complete!")
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

func performHandshake(address string, infoHash []byte) error {
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return fmt.Errorf("handshake connection failed: %v", err)
	}
	defer conn.Close()

	// Send handshake message
	handshakeMsg := fmt.Sprintf("HANDSHAKE:%x\n", infoHash)
	if _, err := conn.Write([]byte(handshakeMsg)); err != nil {
		return fmt.Errorf("failed to send handshake: %v", err)
	}

	// Read handshake response
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read handshake response: %v", err)
	}

	if response != "OK\n" {
		return fmt.Errorf("invalid handshake response: %s", response)
	}

	return nil
}

func checkPieceAvailability(address string, pieceHashes [][]byte) (bool, error) {
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return false, fmt.Errorf("availability check connection failed: %v", err)
	}
	defer conn.Close()

	// Send availability check message
	checkMsg := "CHECK_PIECES\n"
	if _, err := conn.Write([]byte(checkMsg)); err != nil {
		return false, fmt.Errorf("failed to send piece check request: %v", err)
	}

	// Read response
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read piece availability: %v", err)
	}

	return response == "HAVE_PIECES\n", nil
}
