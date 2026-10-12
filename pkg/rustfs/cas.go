package rustfs

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
)

const DefaultChunkSize = 64 * 1024 // 64 KB chunk size

var (
	ErrChunkNotFound = errors.New("chunk not found in CAS store")
	ErrHashMismatch  = errors.New("computed chunk SHA256 does not match declared digest")
)

// ChunkDescriptor identifies a fixed-size content-addressed block.
type ChunkDescriptor struct {
	Offset     int64  `json:"offset"`
	ByteLength int    `json:"byte_length"`
	SHA256     string `json:"sha256"`
}

// CASManifest records the block manifest of a chunked file or evidence blob.
type CASManifest struct {
	RootSHA256 string            `json:"root_sha256"`
	TotalBytes int64             `json:"total_bytes"`
	ChunkSize  int               `json:"chunk_size"`
	Chunks     []ChunkDescriptor `json:"chunks"`
}

// MemoryCAS provides a thread-safe, content-addressed block storage engine.
type MemoryCAS struct {
	mu     sync.RWMutex
	blocks map[string][]byte
}

func NewMemoryCAS() *MemoryCAS {
	return &MemoryCAS{
		blocks: make(map[string][]byte),
	}
}

// PutChunk stores a single block if absent (deduplication happens automatically).
func (c *MemoryCAS) PutChunk(data []byte) (string, error) {
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.blocks[hash]; !exists {
		stored := make([]byte, len(data))
		copy(stored, data)
		c.blocks[hash] = stored
	}
	return hash, nil
}

// GetChunk retrieves a stored block by its SHA-256 digest.
func (c *MemoryCAS) GetChunk(hash string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	block, exists := c.blocks[hash]
	if !exists {
		return nil, ErrChunkNotFound
	}
	res := make([]byte, len(block))
	copy(res, block)
	return res, nil
}

// ChunkReader splits an input stream into 64KB CAS blocks and builds a CASManifest.
func ChunkStream(r io.Reader, cas *MemoryCAS, chunkSize int) (*CASManifest, error) {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}

	manifest := &CASManifest{
		ChunkSize: chunkSize,
		Chunks:    make([]ChunkDescriptor, 0),
	}

	buf := make([]byte, chunkSize)
	rootHasher := sha256.New()
	var currentOffset int64

	for {
		n, err := io.ReadFull(r, buf)
		if n > 0 {
			chunkData := buf[:n]
			rootHasher.Write(chunkData)

			chunkHash, putErr := cas.PutChunk(chunkData)
			if putErr != nil {
				return nil, fmt.Errorf("failed storing chunk at offset %d: %w", currentOffset, putErr)
			}

			manifest.Chunks = append(manifest.Chunks, ChunkDescriptor{
				Offset:     currentOffset,
				ByteLength: n,
				SHA256:     chunkHash,
			})

			currentOffset += int64(n)
			manifest.TotalBytes += int64(n)
		}

		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return nil, err
		}
	}

	manifest.RootSHA256 = hex.EncodeToString(rootHasher.Sum(nil))
	return manifest, nil
}

// ReassembleStream rebuilds the original stream from its chunk manifest.
func ReassembleStream(manifest *CASManifest, cas *MemoryCAS, w io.Writer) error {
	for _, chunk := range manifest.Chunks {
		data, err := cas.GetChunk(chunk.SHA256)
		if err != nil {
			return fmt.Errorf("chunk missing at offset %d (%s): %w", chunk.Offset, chunk.SHA256, err)
		}

		if len(data) != chunk.ByteLength {
			return fmt.Errorf("chunk size mismatch at offset %d: got %d, expected %d", chunk.Offset, len(data), chunk.ByteLength)
		}

		if _, err := io.Copy(w, bytes.NewReader(data)); err != nil {
			return fmt.Errorf("failed writing chunk at offset %d: %w", chunk.Offset, err)
		}
	}
	return nil
}
