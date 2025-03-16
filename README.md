[![GoDoc](https://godoc.org/github.com/KarpelesLab/rest?status.svg)](https://godoc.org/github.com/KarpelesLab/rest)
[![Go Report Card](https://goreportcard.com/badge/github.com/KarpelesLab/rest)](https://goreportcard.com/report/github.com/KarpelesLab/rest)

# KarpelesLab/rest

A comprehensive Go client for interacting with RESTful API services. This package simplifies making HTTP requests to REST endpoints, handling authentication, token renewal, and response parsing.

## Features

- Simple API for RESTful requests with JSON encoding/decoding
- Support for context-based configuration and cancellation
- OAuth2 token management with automatic renewal
- Robust error handling with unwrapping to standard errors
- Generic response parsing with type safety (Go 1.18+)
- Large file uploads with multi-part and AWS S3 support
- Automatic retry on token expiration

## Installation

```bash
go get github.com/KarpelesLab/rest
```

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/KarpelesLab/rest"
)

func main() {
    ctx := context.Background()
    
    // Simple GET request with response as map
    var result map[string]interface{}
    err := rest.Apply(ctx, "Endpoint/Path", "GET", nil, &result)
    if err != nil {
        log.Fatalf("API request failed: %s", err)
    }
    fmt.Printf("Result: %+v\n", result)
    
    // Using the generic As method (Go 1.18+)
    type User struct {
        ID   string `json:"id"`
        Name string `json:"name"`
    }
    
    user, err := rest.As[User](ctx, "Users/Get", "GET", rest.Param{"userId": "123"})
    if err != nil {
        log.Fatalf("Failed to get user: %s", err)
    }
    fmt.Printf("User: %s (%s)\n", user.Name, user.ID)
}
```

## Parameter Order

This library was built based on the original JSON library, where HTTP method was an optional argument. As such it was placed after the path, and this pattern was kept in the Go version. 

While methods like `http.NewRequest` take the method followed by the URL, remembering that the method is sometimes optional in API implementations is a good way to remember the argument order in this library:

```go
rest.Apply(ctx, "Path/Endpoint", "GET", params, &result)
//                  ^path        ^method  ^params ^result
```

## File Uploads

The package provides robust file upload capabilities:

```go
// Upload a file
file, _ := os.Open("largefile.dat")
defer file.Close()

res, err := rest.Upload(ctx, "Files/Upload", "POST", 
    rest.Param{"filename": "myfile.dat"}, file, "application/octet-stream")
if err != nil {
    log.Fatalf("Upload failed: %s", err)
}
```

## Command-line Upload Tool

`restupload` is a command-line tool for uploading large files to specific APIs.

Installation:

```bash
go install github.com/KarpelesLab/rest/cli/restupload@latest
```

## Error Handling

Errors from REST APIs are automatically parsed and can be handled with the standard Go errors package:

```go
_, err := rest.Do(ctx, "Protected/Resource", "GET", nil)
if err != nil {
    if errors.Is(err, os.ErrPermission) {
        // Handle permission denied (403)
    } else if errors.Is(err, fs.ErrNotExist) {
        // Handle not found (404)
    } else {
        // Handle other errors
    }
}
```

## Testing

The package includes a comprehensive test suite that can be run using:

```bash
go test -v github.com/KarpelesLab/rest
```

## License

This package is distributed under the terms of the license found in the [LICENSE](LICENSE) file.

