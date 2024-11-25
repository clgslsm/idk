package torrent

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"os"

	"github.com/jackpal/bencode-go"
)

// Port to listen on
const Port uint16 = 6881

// TorrentFile encodes the metadata from a .torrent file
type TorrentFile struct {
	Announce    string
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	Name        string
}

type bencodeInfo struct {
	Pieces      string `bencode:"pieces"`
	PieceLength int    `bencode:"piece length"`
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
}

type bencodeTorrent struct {
	Announce string      `bencode:"announce"`
	Info     bencodeInfo `bencode:"info"`
}

// Open parses a torrent file
func Open(path string) (TorrentFile, error) {
	fmt.Println(path)
	file, err := os.Open(path)
	if err != nil {
		return TorrentFile{}, err
	}
	defer file.Close()

	bto := bencodeTorrent{}
	err = bencode.Unmarshal(file, &bto)
	if err != nil {
		return TorrentFile{}, err
	}
	fmt.Println(bto.toTorrentFile())
	return bto.toTorrentFile()
}

func (i *bencodeInfo) hash() ([20]byte, error) {
	var buf bytes.Buffer
	err := bencode.Marshal(&buf, *i)
	if err != nil {
		return [20]byte{}, err
	}
	h := sha1.Sum(buf.Bytes())
	return h, nil
}

func (i *bencodeInfo) splitPieceHashes() ([][20]byte, error) {
	hashLen := 20 // Length of SHA-1 hash
	buf := []byte(i.Pieces)
	if len(buf)%hashLen != 0 {
		err := fmt.Errorf("Received malformed pieces of length %d", len(buf))
		return nil, err
	}
	numHashes := len(buf) / hashLen
	hashes := make([][20]byte, numHashes)

	for i := 0; i < numHashes; i++ {
		copy(hashes[i][:], buf[i*hashLen:(i+1)*hashLen])
	}
	return hashes, nil
}

func (bto *bencodeTorrent) toTorrentFile() (TorrentFile, error) {
	infoHash, err := bto.Info.hash()
	if err != nil {
		return TorrentFile{}, err
	}
	pieceHashes, err := bto.Info.splitPieceHashes()
	if err != nil {
		return TorrentFile{}, err
	}
	t := TorrentFile{
		Announce:    bto.Announce,
		InfoHash:    infoHash,
		PieceHashes: pieceHashes,
		PieceLength: bto.Info.PieceLength,
		Length:      bto.Info.Length,
		Name:        bto.Info.Name,
	}
	return t, nil
}

// CreateTorrent builds a TorrentFile from a file path and tracker URL
func CreateTorrent(path string, trackerURL string) (TorrentFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return TorrentFile{}, err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return TorrentFile{}, err
	}

	// Create bencode structs
	bto := bencodeTorrent{
		Announce: trackerURL,
		Info: bencodeInfo{
			PieceLength: 262144, // Standard piece length of 256KB
			Name:        fileInfo.Name(),
			Length:      int(fileInfo.Size()),
		},
	}

	// Calculate pieces hashes
	buf := make([]byte, bto.Info.PieceLength)
	pieces := []byte{}
	for {
		n, err := file.Read(buf)
		if n == 0 {
			break
		}
		if err != nil && err != io.EOF {
			return TorrentFile{}, err
		}
		piece := sha1.Sum(buf[:n])
		pieces = append(pieces, piece[:]...)
	}
	bto.Info.Pieces = string(pieces)

	return bto.toTorrentFile()
}

// Create saves a TorrentFile as a .torrent file
func (t *TorrentFile) createTorrentFile(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	bto := bencodeTorrent{
		Announce: t.Announce,
		Info: bencodeInfo{
			Pieces: string(bytes.Join(func() [][]byte {
				pieces := make([][]byte, len(t.PieceHashes))
				for i := range t.PieceHashes {
					pieces[i] = t.PieceHashes[i][:]
				}
				return pieces
			}(), []byte{})),
			PieceLength: t.PieceLength,
			Length:      t.Length,
			Name:        t.Name,
		},
	}

	return bencode.Marshal(file, bto)
}

func Create(path string) (torrentPath string, err error) {
	trackerURL := "http://localhost:8080/announce"
	torrentFile, err := CreateTorrent(path, trackerURL)
	if err != nil {
		return "", err
	}
	torrentFileName := fmt.Sprintf("%s.torrent", path)
	err = torrentFile.createTorrentFile(torrentFileName)
	if err != nil {
		return "", err
	}
	return torrentFileName, nil
}
