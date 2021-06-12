package rest

import (
	"context"
	"crypto/rand"
	"io"
	"log"
	"testing"
)

func TestUpload(t *testing.T) {
	// input file (512MB, non seekable)
	input := &io.LimitedReader{R: rand.Reader, N: 512 * 1024 * 1024}

	ctx := context.Background()
	res, err := Upload(ctx, "Shell/Bit:upload", "POST", Param{"filename": "test.bin"}, input, "application/octet-stream")

	if err != nil {
		t.Fatalf("failed to do upload: %s", err)
	}

	log.Printf("res = %s", res.Data)
}
