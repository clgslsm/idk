package server

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

// pieceSize = 256KB
const pieceSize = 256 * 1024

// FileWorker handles the file pieces for a specific torrent
type FileWorker struct {
	filePath    string
	pieces      [][]byte
	numPieces   int
	pieceHashes []string
}

// NewFileWorker creates and initializes a new FileWorker
func NewFileWorker(filePath string) (*FileWorker, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("error getting file info: %v", err)
	}

	// Calculate number of pieces
	numPieces := int((fileInfo.Size() + pieceSize - 1) / pieceSize)
	pieces := make([][]byte, numPieces)
	pieceHashes := make([]string, numPieces)

	// Read file into pieces
	for i := 0; i < numPieces; i++ {
		piece := make([]byte, pieceSize)
		n, err := file.ReadAt(piece, int64(i*pieceSize))
		if err != nil && err.Error() != "EOF" {
			return nil, fmt.Errorf("error reading piece %d: %v", i, err)
		}

		// Trim the last piece if needed
		if n < pieceSize {
			piece = piece[:n]
		}

		pieces[i] = piece
		// Calculate hash for the piece
		hash := sha256.Sum256(piece)
		pieceHashes[i] = hex.EncodeToString(hash[:])
	}

	return &FileWorker{
		filePath:    filePath,
		pieces:      pieces,
		numPieces:   numPieces,
		pieceHashes: pieceHashes,
	}, nil
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
	var worker *FileWorker

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
			worker = handleHandshake(conn, message)
			if worker == nil {
				return
			}

		case strings.HasPrefix(message, "Requesting piece:"):
			if worker == nil {
				conn.Write([]byte("ERROR: Handshake required\n"))
				continue
			}
			handlePieceRequest(conn, message, worker)

		default:
			fmt.Printf("Unknown message: %s\n", message)
			conn.Write([]byte("ERROR: Unknown message\n"))
		}
	}
}

func handleHandshake(conn net.Conn, message string) *FileWorker {
	fmt.Printf("Received handshake message: %s\n", message)
	// Get the info hash from the message
	infoHashMessage := strings.TrimPrefix(message, "HANDSHAKE:")
	// Check if the info hash is in the torrent_info.json file
	torrentInfo, err := os.ReadFile("torrent_info.json")
	if err != nil {
		fmt.Printf("Error reading torrent_info.json: %v\n", err)
		return nil
	}
	var torrentInfoMap map[string]string
	err = json.Unmarshal(torrentInfo, &torrentInfoMap)
	if err != nil {
		fmt.Printf("Error unmarshalling torrent_info.json: %v\n", err)
		return nil
	}
	infoHash := torrentInfoMap["InfoHash"]
	if infoHash != infoHashMessage {
		fmt.Printf("Info hash mismatch: %s != %s\n", infoHash, infoHashMessage)
		return nil
	}
	filePath := torrentInfoMap["FilePath"]
	fmt.Printf("File path: %s\n", filePath)
	// Create worker for the file
	worker, err := NewFileWorker(filePath)
	if err != nil {
		fmt.Printf("Error creating file worker: %v\n", err)
		conn.Write([]byte("ERROR: Unable to process file\n"))
		return nil
	}

	conn.Write([]byte("OK\n"))
	return worker
}

func handlePieceRequest(conn net.Conn, message string, worker *FileWorker) {
	fmt.Printf("Received piece request: %s\n", message)

	// Extract the piece index from the message
	// Message format: "Requesting piece: <index>"
	parts := strings.Split(message, ":")
	if len(parts) != 2 {
		conn.Write([]byte("ERROR: Invalid request format\n"))
		return
	}

	index := strings.TrimSpace(parts[1])
	pieceIndex, err := strconv.Atoi(index)
	if err != nil || pieceIndex < 0 || pieceIndex >= worker.numPieces {
		conn.Write([]byte("ERROR: Invalid piece index\n"))
		return
	}

	// Send the piece data to the client
	response := fmt.Sprintf("%s:%s\n", worker.pieceHashes[pieceIndex], string(worker.pieces[pieceIndex]))
	conn.Write([]byte(response))
	fmt.Printf("Sent piece data for index %d\n", pieceIndex)
}
