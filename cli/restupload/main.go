package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"mime"
	"os"
	"path/filepath"

	"github.com/KarpelesLab/rest"
	"github.com/KarpelesLab/webutil"
)

// upload given file(s) to given API

var (
	api    = flag.String("api", "", "endpoint to direct upload to")
	params = flag.String("params", "", "params to pass to the API")
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

	for _, fn := range args {
		log.Printf("Uploading file %s", fn)
		err := doUpload(fn, p)
		if err != nil {
			log.Printf("failed to upload: %s", err)
			os.Exit(1)
		}
	}
}

func doUpload(fn string, p rest.Param) error {
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
	if st, err := f.Stat(); err == nil {
		pCopy["size"] = st.Size()
		pCopy["lastModified"] = st.ModTime().Unix()
	}

	_, err = rest.Upload(context.Background(), *api, "POST", pCopy, f, mimeType)
	return err
}
