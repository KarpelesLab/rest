[![GoDoc](https://godoc.org/github.com/KarpelesLab/rest?status.svg)](https://godoc.org/github.com/KarpelesLab/rest)

# rest API

Allows accessing functions from the rest api.

## method after path?

This library was built based on the original json library, where method was an optional argument. As such it was placed
after the path, and this was kept in the golang version. While methods like `http.NewRequest` take the method followed
by the url, remembering that the method is optional in some implementations is a good way to remember the order of the
arguments here.

# restupload

`restupload` is a nice tool to upload large files to specific APIs.

Installation:

    go install github.com/KarpelesLab/rest/cli/restupload@latest

