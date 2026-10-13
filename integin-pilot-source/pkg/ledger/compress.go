package ledger

import (
	"bytes"
	"io"

	"github.com/klauspost/compress/zstd"
)

// CompressZstd compresses raw ledger audit records for low-bandwidth satellite/VSAT sync.
func CompressZstd(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	enc, err := zstd.NewWriter(&buf)
	if err != nil {
		return nil, err
	}
	if _, err := enc.Write(data); err != nil {
		_ = enc.Close()
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DecompressZstd decompresses a zstd-compressed ledger payload.
func DecompressZstd(compressed []byte) ([]byte, error) {
	dec, err := zstd.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}
	defer dec.Close()
	return io.ReadAll(dec)
}
