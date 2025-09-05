package rest

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"
)

var ErrUploadStalled = errors.New("upload stalled: transferred less than 150KB in 30 seconds")

// stallDetectReader wraps an io.Reader and detects stalls during reading.
// It considers a stall when less than stallThreshold bytes are read in stallTimeout duration.
type stallDetectReader struct {
	reader         io.Reader
	ctx            context.Context
	stallTimeout   time.Duration
	stallThreshold int64
	lastProgress   time.Time
	bytesInPeriod  int64
	closed         chan struct{}
	closeOnce      sync.Once
}

// newStallDetectReader creates a new reader that detects stalls.
// Default: stall if less than 150KB transferred in 30 seconds.
func newStallDetectReader(ctx context.Context, r io.Reader) *stallDetectReader {
	return &stallDetectReader{
		reader:         r,
		ctx:            ctx,
		stallTimeout:   30 * time.Second,
		stallThreshold: 150 * 1024, // 150KB
		lastProgress:   time.Now(),
		closed:         make(chan struct{}),
	}
}

func (sr *stallDetectReader) Read(p []byte) (n int, err error) {

	// Set up a timer for read timeout
	timer := time.NewTimer(sr.stallTimeout)
	defer timer.Stop()

	type result struct {
		n   int
		err error
	}

	readCh := make(chan result, 1)

	// Perform read in goroutine
	go func() {
		n, err := sr.reader.Read(p)
		readCh <- result{n, err}
	}()

	select {
	case <-sr.closed:
		return 0, io.ErrClosedPipe
	case <-sr.ctx.Done():
		return 0, sr.ctx.Err()
	case <-timer.C:
		// Check if we've made enough progress in the period
		if sr.bytesInPeriod < sr.stallThreshold {
			return 0, ErrUploadStalled
		}
		// Reset counter for next period
		sr.bytesInPeriod = 0
		sr.lastProgress = time.Now()
		// Continue waiting for read to complete
		res := <-readCh
		sr.bytesInPeriod += int64(res.n)
		return res.n, res.err
	case res := <-readCh:
		sr.bytesInPeriod += int64(res.n)

		// Check if it's been too long since last progress check
		if time.Since(sr.lastProgress) > sr.stallTimeout {
			if sr.bytesInPeriod < sr.stallThreshold {
				return res.n, ErrUploadStalled
			}
			// Reset for next period
			sr.bytesInPeriod = 0
			sr.lastProgress = time.Now()
		}

		return res.n, res.err
	}
}

// Close implements io.Closer to allow clean shutdown
func (sr *stallDetectReader) Close() error {
	sr.closeOnce.Do(func() {
		close(sr.closed)
	})

	// If the underlying reader implements io.Closer, close it too
	if closer, ok := sr.reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
