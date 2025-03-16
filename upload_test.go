package rest

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"log"
	"testing"
)

type emptyReader struct{}

func (*emptyReader) Read(b []byte) (int, error) {
	return 0, io.EOF
}

// generateTestInput creates a test reader with random data of the specified size
// and returns the reader along with a hash writer that will compute the SHA256
// of all data read from the reader
func generateTestInput(size int64) (io.Reader, hash.Hash) {
	var input io.Reader
	input = &io.LimitedReader{R: rand.Reader, N: size}
	
	// compute sha256 of bytes we send
	hash := sha256.New()
	input = io.TeeReader(input, hash)
	
	return input, hash
}

// TestUpload tests the standard upload mode (put_only=false)
func TestUpload(t *testing.T) {
	// input file (16MB, non seekable)
	input, hash := generateTestInput(16 * 1024 * 1024)

	ctx := context.Background()
	res, err := Upload(ctx, "Misc/Debug:testUpload", "POST", Param{
		"filename": "test.bin",
		"put_only": false,
	}, input, "application/octet-stream")

	if err != nil {
		t.Fatalf("failed to do standard upload: %s", err)
	}

	log.Printf("Standard upload - expected hash = %s", hex.EncodeToString(hash.Sum(nil)))
	log.Printf("Standard upload - response = %s", res.Data)
	
	// Verify we got a Blob__ field in the response
	blobValue, err := res.GetString("Blob__")
	if err != nil || blobValue == "" {
		t.Errorf("Expected Blob__ field in response, got error: %v", err)
	}
}

// TestUploadPutOnly tests the direct PUT upload mode (put_only=true)
func TestUploadPutOnly(t *testing.T) {
	// For put_only=true, we need a seekable reader to determine the size
	// Generate 1MB of random data
	data := make([]byte, 1024*1024)
	if _, err := rand.Read(data); err != nil {
		t.Fatalf("failed to generate random data: %s", err)
	}
	
	// Create a seekable reader
	input := bytes.NewReader(data)
	
	// Calculate expected hash
	hash := sha256.Sum256(data)
	expectedHash := hex.EncodeToString(hash[:])

	ctx := context.Background()
	res, err := Upload(ctx, "Misc/Debug:testUpload", "POST", Param{
		"filename": "test_put_only.bin",
		"put_only": true,
	}, input, "application/octet-stream")

	if err != nil {
		t.Fatalf("failed to do PUT-only upload: %s", err)
	}

	log.Printf("PUT-only upload - expected hash = %s", expectedHash)
	log.Printf("PUT-only upload - response = %s", res.Data)
	
	// Verify we got a Blob__ field in the response
	blobValue, err := res.GetString("Blob__")
	if err != nil || blobValue == "" {
		t.Errorf("Expected Blob__ field in response, got error: %v", err)
	}
	
	// Verify SHA256 matches
	shaValue, err := res.GetString("SHA256")
	if err != nil || shaValue != expectedHash {
		t.Errorf("Expected SHA256 %s, got %s (error: %v)",
			expectedHash, shaValue, err)
	}
}

// TestUploadEmpty tests uploading an empty file in standard mode
func TestUploadEmpty(t *testing.T) {
	// input file (0 bytes, non seekable)
	input := &emptyReader{}

	ctx := context.Background()
	res, err := Upload(ctx, "Misc/Debug:testUpload", "POST", Param{
		"filename": "empty.bin",
		"put_only": false,
	}, input, "application/octet-stream")

	if err != nil {
		t.Fatalf("failed to do empty standard upload: %s", err)
	}

	log.Printf("Empty standard upload - response = %s", res.Data)
	
	// Verify the SHA256 of an empty file
	expectedEmptyHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	
	// Get the SHA256 from the response
	shaValue, err := res.GetString("SHA256")
	if err != nil || shaValue != expectedEmptyHash {
		t.Errorf("Expected SHA256 of empty file to be %s, got %s (error: %v)", 
			expectedEmptyHash, shaValue, err)
	}
}

// TestUploadEmptyPutOnly tests uploading an empty file in PUT-only mode
func TestUploadEmptyPutOnly(t *testing.T) {
	// Create an empty byte slice for a seekable empty reader
	emptyData := make([]byte, 0)
	input := bytes.NewReader(emptyData)

	ctx := context.Background()
	res, err := Upload(ctx, "Misc/Debug:testUpload", "POST", Param{
		"filename": "empty_put_only.bin",
		"put_only": true,
	}, input, "application/octet-stream")

	if err != nil {
		t.Fatalf("failed to do empty PUT-only upload: %s", err)
	}

	log.Printf("Empty PUT-only upload - response = %s", res.Data)
	
	// Verify the SHA256 of an empty file
	expectedEmptyHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	
	// Get the SHA256 from the response
	shaValue, err := res.GetString("SHA256")
	if err != nil || shaValue != expectedEmptyHash {
		t.Errorf("Expected SHA256 of empty file to be %s, got %s (error: %v)", 
			expectedEmptyHash, shaValue, err)
	}
}
