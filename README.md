[![GoDoc](https://godoc.org/github.com/KarpelesLab/rest?status.svg)](https://godoc.org/github.com/KarpelesLab/rest)
[![Go Report Card](https://goreportcard.com/badge/github.com/KarpelesLab/rest)](https://goreportcard.com/report/github.com/KarpelesLab/rest)

# KarpelesLab/rest

A comprehensive Go client for interacting with RESTful API services. This package simplifies making HTTP requests to REST endpoints, handling authentication, token renewal, and response parsing.

## Features

- Simple API for RESTful requests with JSON encoding/decoding
- Support for context-based configuration and cancellation
- Multiple authentication methods:
  - OAuth2 token management with automatic renewal
  - API key authentication with secure request signing
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

## Authentication

### OAuth2 Token Authentication

```go
// Create a token
token := &rest.Token{
    AccessToken: "your-access-token",
    RefreshToken: "your-refresh-token",
    ClientID: "your-client-id",
    Expires: 3600,
}

// Use the token in the context
ctx := token.Use(context.Background())

// Make authenticated requests
result, err := rest.Do(ctx, "Protected/Resource", "GET", nil)
```

### API Key Authentication

```go
// Create an API key with the key ID and secret
apiKey, err := rest.NewApiKey("key-12345", "your-secret")
if err != nil {
    log.Fatalf("Failed to create API key: %v", err)
}

// Use the API key in the context
ctx := apiKey.Use(context.Background())

// Make authenticated requests
// The request will be automatically signed using the API key
result, err := rest.Do(ctx, "Protected/Resource", "GET", nil)
```

```

#### Complete API Key Authentication Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    
    "github.com/KarpelesLab/rest"
)

func main() {
    // Read API key ID and secret from environment
    keyID := os.Getenv("API_KEY_ID")
    keySecret := os.Getenv("API_KEY_SECRET")
    if keyID == "" || keySecret == "" {
        log.Fatal("API_KEY_ID and API_KEY_SECRET environment variables are required")
    }
    
    // Create an API key instance with the key ID and base64-encoded secret
    apiKey, err := rest.NewApiKey(keyID, keySecret)
    if err != nil {
        log.Fatalf("Failed to create API key: %v", err)
    }
    
    // Create a context with the API key
    ctx := context.Background()
    ctx = apiKey.Use(ctx)
    
    // Define the request parameters
    params := map[string]interface{}{
        "limit": 10,
        "filter": "active",
    }
    
    // Make the API call - the request will be automatically signed
    var users []map[string]interface{}
    err = rest.Apply(ctx, "User:list", "GET", params, &users)
    if err != nil {
        log.Fatalf("API request failed: %v", err)
    }
    
    // Process the response
    fmt.Printf("Found %d users\n", len(users))
    for i, user := range users {
        fmt.Printf("User %d: %s (%s)\n", i+1, user["name"], user["email"])
    }
}
```

The authentication process happens automatically:
1. The library adds the API key ID as `_key` parameter
2. It adds the current timestamp as `_time` parameter 
3. It generates a unique nonce as `_nonce` parameter
4. It builds a signature string from the method, path, query parameters, and request body
5. It signs this string with the API key's secret using Ed25519
6. It adds the signature as `_sign` parameter to complete the authentication

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

## API Discovery

To discover available endpoints and their parameters, you can use the `@karpeleslab/klbfw-describe` tool:

```bash
npx @karpeleslab/klbfw-describe SomeEndpoint/Path
```

This will display detailed information about the endpoint, including:
- Available methods
- Required and optional parameters
- Return types
- Access requirements

This is especially useful when exploring new APIs or understanding existing endpoints.

## Testing

The package includes a comprehensive test suite that can be run using:

```bash
go test -v github.com/KarpelesLab/rest
```

## License

This package is distributed under the terms of the license found in the [LICENSE](LICENSE) file.

