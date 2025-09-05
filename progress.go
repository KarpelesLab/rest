package rest

// UploadProgressFunc is a callback function for upload progress updates.
// It receives the number of bytes that have been successfully uploaded.
// The function is called after each part completes uploading.
type UploadProgressFunc func(bytesUploaded int64)
