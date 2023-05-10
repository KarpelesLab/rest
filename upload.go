package rest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type UploadInfo struct {
	// generic
	put  string
	cmpl string
	ctx  context.Context

	MaxPartSize     int64 // maximum size of a single part in MB, defaults to 1024 (1GB)
	ParallelUploads int   // number of parallel uploads to perform (defaults to 3)

	// put upload
	blocksize int64

	// aws upload
	awsid     string
	awskey    string
	awsregion string
	awsname   string
	awshost   string

	awsuploadid string // used during upload
	awstags     []string
	awstagsLk   sync.Mutex
}

type uploadAuth struct {
	Authorization string `json:"authorization"`
}

type uploadAwsResp struct {
	Bucket   string
	Key      string
	UploadId string
}

func Upload(ctx context.Context, req, method string, param Param, f io.Reader, mimeType string) (*Response, error) {
	var upinfo map[string]interface{}

	err := Apply(ctx, req, method, param, &upinfo)
	if err != nil {
		return nil, fmt.Errorf("initial upload query failed: %w", err)
	}

	up, err := PrepareUpload(upinfo)
	if err != nil {
		return nil, fmt.Errorf("upload prepare failed: %w", err)
	}

	ln := int64(-1)

	if fs, ok := f.(io.Seeker); ok {
		ln, err = fs.Seek(0, io.SeekEnd)
		if err != nil {
			// seek failed, let's continue in the unknown
			ln = -1
		} else {
			// seek back to the start
			fs.Seek(0, io.SeekStart)
		}
	}

	return up.Do(ctx, f, mimeType, ln)
}

// upload for platform files
func PrepareUpload(req map[string]interface{}) (*UploadInfo, error) {
	// we have the following parameters:
	// * PUT (url to put to)
	// * Complete (APÏ to call upon completion)
	// we optionally support multipart upload for images over 5GB through extra parameters

	up := &UploadInfo{
		MaxPartSize:     1024,
		ParallelUploads: 3,
	}
	if err := up.parse(req); err != nil {
		return nil, err
	}

	return up, nil
}

func (u *UploadInfo) String() string {
	return u.put
}

func (u *UploadInfo) parse(req map[string]interface{}) error {
	var ok bool

	//log.Printf("parsing upload response: %+v", req)

	// strict minimum: PUT & Complete
	u.put, ok = req["PUT"].(string)
	if !ok {
		return errors.New("required parameter PUT not found")
	}
	u.cmpl, ok = req["Complete"].(string)
	if !ok {
		return errors.New("required parameter Complete not found")
	}

	// vars we care about:
	// * Cloud_Aws_Bucket_Upload__
	// * Key
	// * Bucket_Endpoint.Region
	// * Bucket_Endpoint.Name
	// * Bucket_Endpoint.Host

	// if we can't grab any of these, drop the whole thing and not set u.awsid so it won't be used

	id, ok := req["Cloud_Aws_Bucket_Upload__"].(string)
	if !ok {
		// no id, but we don't care
		if bs, ok := req["Blocksize"].(float64); ok {
			// we got a blocksize, this uses the new upload method
			u.blocksize = int64(bs)
			return nil
		}
		return nil
	}
	bucket, ok := req["Bucket_Endpoint"].(map[string]interface{})
	if !ok {
		return nil
	}
	u.awskey, ok = req["Key"].(string)
	if !ok {
		return nil
	}
	u.awsregion, ok = bucket["Region"].(string)
	if !ok {
		return nil
	}
	u.awsname = bucket["Name"].(string)
	if !ok {
		return nil
	}
	u.awshost = bucket["Host"].(string)
	if !ok {
		return nil
	}
	// all ok, set awsid
	u.awsid = id

	return nil
}

func (u *UploadInfo) Do(ctx context.Context, f io.Reader, mimeType string, ln int64) (*Response, error) {
	u.ctx = ctx

	if u.blocksize > 0 {
		return u.partUpload(f, mimeType)
	}
	if u.awsid != "" {
		if ln == -1 || ln > 64*1024*1024 {
			return u.awsUpload(f, mimeType)
		}
	}

	if ln == -1 || ln > 5*1024*1024*1024 {
		return nil, errors.New("cannot upload using PUT method without a known length of less than 5GB")
	}

	// we can use simple PUT
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u.put, f)
	if err != nil {
		return nil, err
	}

	req.ContentLength = ln
	req.Header.Set("Content-Type", mimeType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() // avoid leaking stuff
	// read full response, discard (ensures upload completed)
	io.Copy(ioutil.Discard, resp.Body)

	return u.complete()
}

func (u *UploadInfo) complete() (*Response, error) {
	return Do(u.ctx, u.cmpl, "POST", map[string]interface{}{})
}

func (u *UploadInfo) partUpload(f io.Reader, mimeType string) (*Response, error) {
	// partUpload works similar to awsUpload but when uploading to the new kind of PUT server

	// let's upload
	partNo := 0
	errCh := make(chan error, 2) // enough just in case
	nwg := newNWG()

	eof := false
	for !eof {
		nwg.Wait(u.ParallelUploads - 1)
		partNo += 1

		readCh := make(chan error)

		nwg.Add(1)
		go u.partUploadPart(f, mimeType, partNo, readCh, errCh, nwg)

		select {
		case err := <-readCh:
			if err == io.EOF {
				eof = true
			} else if err != nil {
				// fatal error
				return nil, err
			}
		case err := <-errCh:
			// fatal error
			return nil, err
		}
	}

	// wait for nwg completion
	go func() {
		nwg.Wait(0)
		// send "no error"
		select {
		case errCh <- nil:
		default:
			// do not wait if send fails
		}
	}()

	// read & check error (cause waiting for completion)
	err := <-errCh
	if err != nil {
		// fatal error
		return nil, err
	}

	// finalize
	return u.complete()
}

func (u *UploadInfo) partUploadPart(f io.Reader, mimeType string, partNo int, readCh, errCh chan<- error, nwg *numeralWaitGroup) {
	// prepare to upload a part
	defer nwg.Done()

	// we use temp files as to avoid using too much memory
	tmpf, err := ioutil.TempFile("", "upload*.bin")
	if err != nil {
		// failed to create temp file
		readCh <- err
		return
	}
	// cleanup
	defer func() {
		tmpf.Close()
		os.Remove(tmpf.Name())
	}()

	n, err := io.CopyN(tmpf, f, u.blocksize)
	if err != nil {
		if err != io.EOF {
			// fatal error
			errCh <- err
			return
		}
		readCh <- err
		if n == 0 {
			return
		}
	} else if n == 0 {
		// no data to upload, just return EOF
		readCh <- io.EOF
		return
	} else {
		// end of read
		readCh <- nil
	}

	// rewind tmpf
	tmpf.Seek(0, io.SeekStart)

	// we can use simple PUT
	req, err := http.NewRequestWithContext(u.ctx, http.MethodPut, u.put, tmpf)
	if err != nil {
		select {
		case errCh <- err:
		default:
		}
		return
	}

	start := int64(partNo-1) * u.blocksize
	end := start + n - 1 // inclusive

	req.ContentLength = n // from io.CopyN
	req.Header.Set("Content-Type", mimeType)
	req.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/*", start, end))

	// perform upload
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		select {
		case errCh <- err:
		default:
		}
		return
	}
	defer resp.Body.Close() // avoid leaking stuff
	// read full response, discard (ensures upload completed)
	_, err = io.Copy(ioutil.Discard, resp.Body)
	if err != nil {
		select {
		case errCh <- err:
		default:
		}
		return
	}
}

func (u *UploadInfo) awsUpload(f io.Reader, mimeType string) (*Response, error) {
	// awsUpload is a magic method that does not need to know upload length as it will split file into manageable sized pieces.
	err := u.awsInit(mimeType)
	if err != nil {
		return nil, err
	}

	// let's upload
	partNo := 0
	errCh := make(chan error, 2) // enough just in case
	nwg := newNWG()

	eof := false
	for !eof {
		nwg.Wait(u.ParallelUploads - 1)
		partNo += 1

		readCh := make(chan error)

		nwg.Add(1)
		go u.awsUploadPart(f, partNo, readCh, errCh, nwg)

		select {
		case err := <-readCh:
			if err == io.EOF {
				eof = true
			} else if err != nil {
				// fatal error, give up
				u.awsAbort()
				return nil, err
			}
		case err := <-errCh:
			// fatal error, give up
			u.awsAbort()
			return nil, err
		}
	}

	// wait for nwg completion
	go func() {
		nwg.Wait(0)
		// send "no error"
		select {
		case errCh <- nil:
		default:
			// do not wait if send fails
		}
	}()

	// read & check error (cause waiting for completion)
	err = <-errCh
	if err != nil {
		// fatal error
		u.awsAbort()
		return nil, err
	}

	// finalize
	err = u.awsFinalize()
	if err != nil {
		return nil, err
	}

	return u.complete()
}

func (u *UploadInfo) awsFinalize() error {
	// see https://docs.aws.amazon.com/AmazonS3/latest/API/mpUploadComplete.html
	buf := &bytes.Buffer{}

	fmt.Fprintf(buf, "<CompleteMultipartUpload>")
	for n, tag := range u.awstags {
		fmt.Fprintf(buf, "<Part><PartNumber>%d</PartNumber><ETag>%s</ETag></Part>", n+1, tag)
	}
	fmt.Fprintf(buf, "</CompleteMultipartUpload>")

	resp, err := u.awsReq("POST", "uploadId="+u.awsuploadid, bytes.NewReader(buf.Bytes()), http.Header{"Content-Type": []string{"text/xml"}})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(ioutil.Discard, resp.Body)

	return err
}

func (u *UploadInfo) awsUploadPart(f io.Reader, partNo int, readCh, errCh chan<- error, nwg *numeralWaitGroup) {
	// prepare to upload a part
	defer nwg.Done()

	// maxLen in MB
	maxLen := int64(partNo * 64)
	if maxLen > u.MaxPartSize {
		maxLen = u.MaxPartSize
	}
	if maxLen < 5 {
		// minimum size enforced by aws (except for last part)
		maxLen = 5
	}

	tmpf, err := ioutil.TempFile("", "upload*.bin")
	if err != nil {
		// failed to create temp file
		readCh <- err
		return
	}
	// cleanup
	defer func() {
		tmpf.Close()
		os.Remove(tmpf.Name())
	}()

	n, err := io.CopyN(tmpf, f, maxLen*1024*1024)
	if err != nil {
		if err != io.EOF {
			// fatal error
			errCh <- err
			return
		}
		readCh <- err
		if n == 0 && partNo != 1 {
			return
		}
	} else if n == 0 && partNo != 1 {
		// no data to upload, just return EOF unless we are part #1
		readCh <- io.EOF
		return
	} else {
		// end of read
		readCh <- nil
	}

	// need to upload to aws
	resp, err := u.awsReq("PUT", fmt.Sprintf("partNumber=%d&uploadId=%s", partNo, u.awsuploadid), tmpf, nil)
	if err != nil {
		select {
		case errCh <- err:
		default:
		}
		return
	}
	defer resp.Body.Close()
	_, err = io.Copy(ioutil.Discard, resp.Body)
	if err != nil {
		select {
		case errCh <- err:
		default:
		}
		return
	}

	// store etag value
	u.setTag(partNo, resp.Header.Get("Etag"))
}

func (u *UploadInfo) setTag(partNo int, tag string) {
	u.awstagsLk.Lock()
	defer u.awstagsLk.Unlock()

	pos := partNo - 1

	if cap(u.awstags) <= pos {
		// need to increase cap
		tmp := make([]string, len(u.awstags), cap(u.awstags)+64)
		copy(tmp, u.awstags)
		u.awstags = tmp
	}

	if pos >= len(u.awstags) {
		u.awstags = u.awstags[:pos+1]
	}
	u.awstags[pos] = tag
}

func (u *UploadInfo) awsAbort() error {
	resp, err := u.awsReq("DELETE", "uploadId="+u.awsuploadid, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(ioutil.Discard, resp.Body)
	return err
}

func (u *UploadInfo) awsInit(mimeType string) error {
	// see: https://docs.aws.amazon.com/AmazonS3/latest/API/API_CreateMultipartUpload.html
	resp, err := u.awsReq("POST", "uploads=", nil, http.Header{"Content-Type": []string{mimeType}, "X-Amz-Acl": []string{"private"}})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	dec := xml.NewDecoder(resp.Body)

	res := &uploadAwsResp{}
	err = dec.Decode(res)
	if err != nil {
		return err
	}

	if res.UploadId == "" {
		return errors.New("failed to read aws upload id")
	}

	u.awsuploadid = res.UploadId
	return nil
}

func (u *UploadInfo) awsReq(method, query string, body io.ReadSeeker, headers http.Header) (*http.Response, error) {
	if headers == nil {
		headers = http.Header{}
	}

	// seek at end to know length
	var ln int64
	if body != nil {
		var err error
		ln, err = body.Seek(0, io.SeekEnd)
		if err != nil {
			return nil, err
		}
		body.Seek(0, io.SeekStart)
	}

	// perform aws request using remote signature
	var bodyHash string
	if ln == 0 {
		bodyHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" // sha256("")
	} else {
		h := sha256.New()
		_, err := io.Copy(h, body)
		if err != nil {
			return nil, err
		}
		body.Seek(0, io.SeekStart) // reset to beginning

		bodyHash = hex.EncodeToString(h.Sum(nil))
	}

	ts := time.Now().UTC().Format("20060102T150405Z") // amz format
	tsD := ts[:8]                                     // YYYYMMDD

	headers.Set("X-Amz-Content-Sha256", bodyHash)
	headers.Set("X-Amz-Date", ts)

	awsAuthStr := []string{
		"AWS4-HMAC-SHA256",
		ts,
		tsD + "/" + u.awsregion + "/s3/aws4_request",
		method,
		"/" + u.awsname + "/" + u.awskey,
		query,
		"host:" + u.awshost,
	}

	// list headers to sign (host and anything starting with x-)
	signHead := []string{"host"}
	for k, _ := range headers {
		s := strings.ToLower(k)
		if strings.HasPrefix(s, "x-") {
			signHead = append(signHead, s)
		}
	}

	// sort signHead
	sort.Strings(signHead)

	// add strings
	for _, h := range signHead {
		if h == "host" {
			// already added
			continue
		}
		awsAuthStr = append(awsAuthStr, h+":"+headers.Get(h))
	}
	awsAuthStr = append(awsAuthStr, "")
	awsAuthStr = append(awsAuthStr, strings.Join(signHead, ";"))
	awsAuthStr = append(awsAuthStr, bodyHash)

	// generate signature
	auth := &uploadAuth{}
	err := Apply(u.ctx, "Cloud/Aws/Bucket/Upload/"+u.awsid+":signV4", "POST", Param{"headers": strings.Join(awsAuthStr, "\n")}, auth)
	if err != nil {
		return nil, err
	}
	headers.Set("Authorization", auth.Authorization)

	// perform the query
	target := "https://" + u.awshost + "/" + u.awsname + "/" + u.awskey
	if query != "" {
		target += "?" + query
	}
	req, err := http.NewRequestWithContext(u.ctx, method, target, body)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header[k] = v
	}

	if ln > 0 {
		req.ContentLength = ln
	} else {
		req.Header.Set("Content-Length", "0")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed: %s\ndetails: %s", resp.Status, body)
	}
	return resp, err
}
