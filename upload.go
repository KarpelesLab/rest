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
	"time"
)

type UploadInfo struct {
	// generic
	put  string
	cmpl string

	// aws upload
	awsid     string
	awskey    string
	awsregion string
	awsname   string
	awshost   string

	awsuploadid string // used during upload
	awstags     []string
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
		return nil, err
	}

	up, err := PrepareUpload(upinfo)
	if err != nil {
		return nil, err
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
	// we will need to support multipart upload for images over 5GB but this turns out to be fairly complex, and won't be needed after we switch away from S3.

	up := &UploadInfo{}
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
	if u.awsid != "" {
		if ln == -1 || ln > 64*1024*1024 {
			return u.awsUpload(ctx, f, mimeType)
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

	return u.complete(ctx)
}

func (u *UploadInfo) complete(ctx context.Context) (*Response, error) {
	return Do(ctx, u.cmpl, "POST", map[string]interface{}{})
}

func (u *UploadInfo) awsUpload(ctx context.Context, f io.Reader, mimeType string) (*Response, error) {
	// awsUpload is a magic method that does not need to know upload length as it will split file into manageable sized pieces.
	err := u.awsInit(ctx, mimeType)
	if err != nil {
		return nil, err
	}

	// let's upload
	partNo := 0

	for {
		partNo += 1
		err = u.awsUploadPart(ctx, f, partNo)
		if err != nil {
			if err == io.EOF {
				// completed
				break
			}
			// another error → give up
			u.awsAbort(ctx)
			return nil, err
		}
	}

	err = u.awsFinalize(ctx)
	if err != nil {
		return nil, err
	}

	return u.complete(ctx)
}

func (u *UploadInfo) awsFinalize(ctx context.Context) error {
	// see https://docs.aws.amazon.com/AmazonS3/latest/API/mpUploadComplete.html
	buf := &bytes.Buffer{}

	fmt.Fprintf(buf, "<CompleteMultipartUpload>")
	for n, tag := range u.awstags {
		fmt.Fprintf(buf, "<Part><PartNumber>%d</PartNumber><ETag>%s</ETag></Part>", n+1, tag)
	}
	fmt.Fprintf(buf, "</CompleteMultipartUpload>")

	resp, err := u.awsReq(ctx, "POST", "uploadId="+u.awsuploadid, bytes.NewReader(buf.Bytes()), http.Header{"Content-Type": []string{"text/xml"}})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(ioutil.Discard, resp.Body)

	return err
}

func (u *UploadInfo) awsUploadPart(ctx context.Context, f io.Reader, partNo int) error {
	// prepare to upload a part

	// maxLen in MB
	maxLen := int64(partNo * 64)
	if maxLen > 1024 {
		maxLen = 1024
	}

	tmpf, err := ioutil.TempFile("", "upload*.bin")
	if err != nil {
		return err
	}
	// cleanup
	defer func() {
		tmpf.Close()
		os.Remove(tmpf.Name())
	}()

	eof := false
	n, err := io.CopyN(tmpf, f, maxLen*1024*1024)
	if err != nil {
		if err == io.EOF {
			eof = true
		} else {
			return err
		}
	}
	if n == 0 {
		// no data to upload, just return EOF
		return io.EOF
	}

	// need to upload to aws
	resp, err := u.awsReq(ctx, "PUT", fmt.Sprintf("partNumber=%d&uploadId=%s", partNo, u.awsuploadid), tmpf, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(ioutil.Discard, resp.Body)
	if err != nil {
		return err
	}

	// store etag value
	u.awstags = append(u.awstags, resp.Header.Get("Etag"))

	if eof {
		return io.EOF
	}
	return nil
}

func (u *UploadInfo) awsAbort(ctx context.Context) error {
	resp, err := u.awsReq(ctx, "DELETE", "uploadId="+u.awsuploadid, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(ioutil.Discard, resp.Body)
	return err
}

func (u *UploadInfo) awsInit(ctx context.Context, mimeType string) error {
	// see: https://docs.aws.amazon.com/AmazonS3/latest/API/API_CreateMultipartUpload.html
	resp, err := u.awsReq(ctx, "POST", "uploads=", nil, http.Header{"Content-Type": []string{mimeType}, "X-Amz-Acl": []string{"private"}})
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

func (u *UploadInfo) awsReq(ctx context.Context, method, query string, body io.ReadSeeker, headers http.Header) (*http.Response, error) {
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
	err := Apply(ctx, "Cloud/Aws/Bucket/Upload/"+u.awsid+":signV4", "POST", Param{"headers": strings.Join(awsAuthStr, "\n")}, auth)
	if err != nil {
		return nil, err
	}
	headers.Set("Authorization", auth.Authorization)

	// perform the query
	target := "https://" + u.awshost + "/" + u.awsname + "/" + u.awskey
	if query != "" {
		target += "?" + query
	}
	req, err := http.NewRequestWithContext(ctx, method, target, body)
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
		resp.Body.Close()
		return nil, fmt.Errorf("request failed: %s", resp.Status)
	}
	return resp, err
}
