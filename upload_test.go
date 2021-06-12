package rest

import (
	"context"
	"crypto/rand"
	"io"
	"testing"
)

func TestUpload(t *testing.T) {
	// test
	var params map[string]interface{}

	ctx := context.Background()
	err := Apply(ctx, "Shell/Bit:upload", "POST", Param{"filename": "test.bin"}, &params)
	if err != nil {
		t.Fatalf("failed to init upload: %s", err)
	}

	up, err := PrepareUpload(params)
	if err != nil {
		t.Fatalf("failed to prepare upload: %s", err)
	}

	// input file (512MB)
	input := &io.LimitedReader{R: rand.Reader, N: 512 * 1024 * 1024}

	err = up.Do(ctx, input, "application/octet-stream", -1)
	if err != nil {
		t.Fatalf("failed to do upload: %s", err)
	}
}
