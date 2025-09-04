package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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

// progressReader wraps an io.Reader and tracks progress
type progressReader struct {
	reader    io.Reader
	total     int64
	current   int64
	fileName  string
	lastPrint time.Time
}

func newProgressReader(r io.Reader, total int64, fileName string) *progressReader {
	return &progressReader{
		reader:   r,
		total:    total,
		fileName: fileName,
	}
}

func (pr *progressReader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)
	pr.current += int64(n)

	// Update progress display every 100ms
	now := time.Now()
	if now.Sub(pr.lastPrint) >= 100*time.Millisecond {
		pr.displayProgress()
		pr.lastPrint = now
	}

	// Display final progress on completion
	if err == io.EOF {
		pr.displayProgress()
		fmt.Fprintf(os.Stderr, "\027[K\n")
	}

	return n, err
}

func (pr *progressReader) displayProgress() {
	if pr.total <= 0 {
		// Unknown total size, show bytes uploaded
		fmt.Fprintf(os.Stderr, "\r%s: %s uploaded\027[K", pr.fileName, formatBytes(pr.current))
		return
	}

	// Calculate percentage
	percent := float64(pr.current) * 100.0 / float64(pr.total)

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
	fmt.Fprintf(os.Stderr, "\r%s: [%s] %.1f%% (%s/%s)\027[K",
		pr.fileName,
		string(bar),
		percent,
		formatBytes(pr.current),
		formatBytes(pr.total))
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

	// Create reader with progress tracking if needed
	var reader io.Reader = f
	if showProgress {
		reader = newProgressReader(f, fileSize, filepath.Base(fn))
	}

	_, err = rest.Upload(ctx, *api, *method, pCopy, reader, mimeType)

	if showProgress && err == nil {
		// Ensure we end with a newline after progress display
		fmt.Fprintf(os.Stderr, "\027[K\n")
	}

	return err
}
