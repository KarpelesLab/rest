package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/KarpelesLab/rest"
	"github.com/KarpelesLab/webutil"
	"golang.org/x/term"
)

// upload given file(s) to given API

// requestHandler is a context wrapper that intercepts *http.Request values to modify them
type requestHandler struct {
	context.Context
	cookies   string
	getParams string
}

// Value intercepts *http.Request values to add cookies and GET parameters
func (rh *requestHandler) Value(key any) any {
	if req, ok := key.(*http.Request); ok {
		// Add cookies to the request
		if rh.cookies != "" {
			req.Header.Set("Cookie", rh.cookies)
		}
		// Add GET parameters to the URL
		if rh.getParams != "" {
			if req.URL.RawQuery != "" {
				req.URL.RawQuery += "&" + rh.getParams
			} else {
				req.URL.RawQuery = rh.getParams
			}
		}
		return nil
	}
	// Pass through to parent context for other values
	return rh.Context.Value(key)
}

var (
	api       = flag.String("api", "", "endpoint to direct upload to")
	params    = flag.String("params", "", "params to pass to the API")
	getParams = flag.String("get", "", "GET query string parameters to append to the URL")
	quiet     = flag.Bool("quiet", false, "suppress progress output")
	hostname  = flag.String("hostname", "", "override API hostname (e.g., api.example.com)")
	method    = flag.String("method", "POST", "HTTP method for the initial API request")
	cookies   = flag.String("cookies", "", "cookies to send with the request (format: name1=value1; name2=value2)")
)

func main() {
	flag.Parse()
	if *api == "" {
		log.Printf("parameter -api is required")
		flag.Usage()
		os.Exit(1)
	}

	var p rest.Param = make(map[string]any)

	if param := *params; param != "" {
		if param[0] == '{' {
			// json
			json.Unmarshal([]byte(param), &p)
		} else {
			// url encoded
			p = webutil.ParsePhpQuery(param)
		}
	}

	args := flag.Args()

	// Prepare context with hostname override if provided
	ctx := context.Background()
	if *hostname != "" {
		backendURL := &url.URL{
			Scheme: "https",
			Host:   *hostname,
		}
		ctx = context.WithValue(ctx, rest.BackendURL, backendURL)
	}

	// Wrap context with request handler if cookies or GET params are provided
	if *cookies != "" || *getParams != "" {
		ctx = &requestHandler{
			Context:   ctx,
			cookies:   *cookies,
			getParams: *getParams,
		}
	}

	// Check if stderr is a terminal
	showProgress := !*quiet && term.IsTerminal(int(os.Stderr.Fd()))

	for _, fn := range args {
		if !showProgress {
			log.Printf("Uploading file %s", fn)
		}
		err := doUpload(ctx, fn, p, showProgress)
		if err != nil {
			log.Printf("failed to upload: %s", err)
			os.Exit(1)
		}
	}
}

// progressTracker tracks upload progress using a callback
type progressTracker struct {
	total     int64
	current   int64
	fileName  string
	lastPrint time.Time
	mu        sync.Mutex
}

func newProgressTracker(total int64, fileName string) *progressTracker {
	return &progressTracker{
		total:    total,
		fileName: fileName,
	}
}

func (pt *progressTracker) updateProgress(bytesUploaded int64) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.current += bytesUploaded

	// Update progress display every 100ms
	now := time.Now()
	if now.Sub(pt.lastPrint) >= 100*time.Millisecond {
		pt.displayProgress()
		pt.lastPrint = now
	}
}

func (pt *progressTracker) displayProgress() {
	if pt.total <= 0 {
		// Unknown total size, show bytes uploaded
		fmt.Fprintf(os.Stderr, "\r%s: %s uploaded\033[K", pt.fileName, formatBytes(pt.current))
		return
	}

	// Calculate percentage
	percent := float64(pt.current) * 100.0 / float64(pt.total)

	// Create progress bar
	barWidth := 30
	filled := int(percent * float64(barWidth) / 100)
	if filled > barWidth {
		filled = barWidth
	}

	bar := make([]rune, barWidth)
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar[i] = '█'
		} else {
			bar[i] = '░'
		}
	}

	// Display progress
	fmt.Fprintf(os.Stderr, "\r%s: [%s] %.1f%% (%s/%s)\033[K",
		pt.fileName,
		string(bar),
		percent,
		formatBytes(pt.current),
		formatBytes(pt.total))
}

func (pt *progressTracker) finish() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.displayProgress()
	fmt.Fprintf(os.Stderr, "\033[K\n")
}

func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func doUpload(ctx context.Context, fn string, p rest.Param, showProgress bool) error {
	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer f.Close()

	mimeType := mime.TypeByExtension(filepath.Ext(fn))

	var pCopy rest.Param = make(map[string]any)
	for k, v := range p {
		pCopy[k] = v
	}
	pCopy["filename"] = filepath.Base(fn)
	pCopy["type"] = mimeType

	var fileSize int64
	if st, err := f.Stat(); err == nil {
		fileSize = st.Size()
		pCopy["size"] = fileSize
		pCopy["lastModified"] = st.ModTime().Unix()
	}

	// Setup progress tracking if needed
	var tracker *progressTracker
	if showProgress {
		tracker = newProgressTracker(fileSize, filepath.Base(fn))
		// Add the progress callback to the context
		ctx = context.WithValue(ctx, rest.UploadProgress, rest.UploadProgressFunc(tracker.updateProgress))
	}

	_, err = rest.Upload(ctx, *api, *method, pCopy, f, mimeType)

	if showProgress && tracker != nil {
		tracker.finish()
	}

	return err
}
