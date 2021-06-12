package rest

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"testing"
)

func TestUpload(t *testing.T) {
	// input file (512MB, non seekable)
	var input io.Reader
	input = &io.LimitedReader{R: rand.Reader, N: 512 * 1024 * 1024}
	// compute sha256 of bytes we send
	hash := sha256.New()
	input = io.TeeReader(input, hash)

	ctx := context.Background()
	res, err := Upload(ctx, "Shell/Bit:upload", "POST", Param{"filename": "test.bin"}, input, "application/octet-stream")

	if err != nil {
		t.Fatalf("failed to do upload: %s", err)
	}

	log.Printf("expected hash = %s", hex.EncodeToString(hash.Sum(nil)))
	log.Printf("res = %s", res.Data)
}
