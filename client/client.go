package client

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"tcp-app/torrent"
)

type PieceWork struct {
	Index int
	Hash  []byte
	Size  int64
}

type PieceResult struct {
	Index int
	Data  string
	Error error
}

func StartDownload(torrentFile string) {
	fmt.Println("Starting download for:", torrentFile)

	// Parse torrent file using the torrent package
	tf, err := torrent.Open(torrentFile)
	if err != nil {
		fmt.Printf("Error opening torrent file: %v\n", err)
		return
	}

	// Mock the list of peers
	peers := []string{"192.168.68.151:8080"}

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

		activePeers = append(activePeers, peer)
	}

	if len(activePeers) == 0 {
		fmt.Println("No available peers found!")
		return
	}

	// Create channels for the worker pool
	const numWorkers = 3
	workQueue := make(chan PieceWork, len(tf.PieceHashes))
	results := make(chan PieceResult, len(tf.PieceHashes))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			downloadWorker(activePeers[0], workQueue, results)
		}()
	}

	// Queue pieces to download
	sizeToDownload := tf.Length
	for i, pieceHash := range tf.PieceHashes {
		sizePieceToDownload := min(tf.PieceLength, sizeToDownload)
		sizeToDownload -= sizePieceToDownload

		workQueue <- PieceWork{
			Index: i,
			Hash:  pieceHash[:],
			Size:  int64(sizePieceToDownload),
		}
	}
	close(workQueue)

	// Wait for workers to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results and merge file
	piecesByIndex := make(map[int]string)
	for result := range results {
		if result.Error != nil {
			fmt.Printf("Error downloading piece %d: %v\n", result.Index, result.Error)
			continue
		}
		piecesByIndex[result.Index] = result.Data
		fmt.Printf("Successfully downloaded piece %d\n", result.Index)
	}

	// Merge pieces into final file
	if err := mergePieces(tf.Name, piecesByIndex, len(tf.PieceHashes)); err != nil {
		fmt.Printf("Error merging pieces: %v\n", err)
		return
	}

	fmt.Println("Download complete!")
}

func downloadWorker(peer string, work <-chan PieceWork, results chan<- PieceResult) {
	for piece := range work {
		data, err := requestPieceFromPeer(peer, fmt.Sprintf("%d:%x:%d",
			piece.Index, piece.Hash, piece.Size))

		results <- PieceResult{
			Index: piece.Index,
			Data:  data,
			Error: err,
		}
	}
}

func mergePieces(fileName string, pieces map[int]string, numPieces int) error {
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	// Write pieces in order
	for i := 0; i < numPieces; i++ {
		data, exists := pieces[i]
		if !exists {
			return fmt.Errorf("missing piece %d", i)
		}
		if _, err := file.WriteString(data); err != nil {
			return fmt.Errorf("failed to write piece %d: %v", i, err)
		}
	}

	return nil
}

func requestPieceFromPeer(address, message string) (string, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return "", fmt.Errorf("error connecting to peer: %v", err)
	}
	defer conn.Close()

	message = "Requesting piece: " + message
	// Send request for the piece
	_, err = conn.Write([]byte(message))
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}

	// Read response from the peer
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	fmt.Printf("Response from peer: %s", response)
	return response, nil
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
