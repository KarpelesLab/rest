package rest

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// ApiKey represents an API key with its secret for signing requests.
// It contains the key ID and secret key used for request signing.
type ApiKey struct {
	KeyID     string
	SecretKey []byte
}

// apiKeyValue is a type used as a context key for API key storage.
type apiKeyValue int

// withApiKey is a context wrapper that holds an API key value.
type withApiKey struct {
	context.Context
	apiKey *ApiKey
}

// No special error variables needed after removing ParseKey

// Value implements the context.Context Value method for withApiKey.
// It returns the API key for apiKeyValue keys and delegates to the parent context otherwise.
func (w *withApiKey) Value(v any) any {
	if _, ok := v.(apiKeyValue); ok {
		return w.apiKey
	}

	return w.Context.Value(v)
}

// NewApiKey creates a new ApiKey instance from separate key ID and secret parameters.
// This is the recommended way to create an ApiKey instance.
//
// Parameters:
// - keyID: The API key identifier
// - secret: The secret key as a base64-encoded string
//
// Returns:
// - An ApiKey instance and nil if successful
// - nil and an error if the secret cannot be properly decoded or is invalid
//
// The secret is a base64url-encoded (using - and _ characters) Ed25519 private key provided by the server.
func NewApiKey(keyID, secret string) (*ApiKey, error) {
	// Decode the secret as base64url (which uses - and _ instead of + and /)
	decodedSecret, err := base64.RawURLEncoding.DecodeString(secret)
	if err != nil {
		// Try standard base64 as fallback
		decodedSecret, err = base64.StdEncoding.DecodeString(secret)
		if err != nil {
			return nil, fmt.Errorf("invalid base64 secret: %w", err)
		}
	}

	// The API expects the secret key to be a full Ed25519 private key (64 bytes)
	// We'll use the raw key as provided by the server
	secretKey := decodedSecret

	return &ApiKey{
		KeyID:     keyID,
		SecretKey: secretKey,
	}, nil
}

// Note: ParseKey function has been removed in favor of directly using NewApiKey

// Use returns a new context that includes this API key for authentication.
// The API key will be used for all REST API calls that use this context.
func (a *ApiKey) Use(ctx context.Context) context.Context {
	return &withApiKey{ctx, a}
}

// generateSignature creates a signature for a REST API request.
// It builds a signing string from the method, path, query string, and request body hash.
func (a *ApiKey) generateSignature(method, path string, queryParams url.Values, body []byte) (string, error) {
	// Generate a SHA256 hash of the request body
	h := sha256.New()
	if body != nil {
		h.Write(body)
	}
	bodyHash := h.Sum(nil)

	// We need to create a copy of queryParams without the _sign parameter
	queryParamsCopy := url.Values{}
	for k, v := range queryParams {
		if k != "_sign" {
			queryParamsCopy[k] = v
		}
	}

	// Create the signing string with null character separators
	var signString bytes.Buffer
	signString.WriteString(method)
	signString.WriteByte(0)
	signString.WriteString(path)
	signString.WriteByte(0)
	signString.WriteString(queryParamsCopy.Encode())
	signString.WriteByte(0)
	signString.Write(bodyHash)

	// Sign the string using Ed25519
	signature := ed25519.Sign(a.SecretKey, signString.Bytes())

	// Encode the signature in base64url format
	encoded := base64.RawURLEncoding.EncodeToString(signature)

	return encoded, nil
}

// applyParams adds API key parameters to the query parameters, including key, nonce, time, and signature.
func (a *ApiKey) applyParams(ctx context.Context, method, path string, queryParams url.Values, body []byte) error {
	if a == nil {
		return fmt.Errorf("nil API key")
	}

	// Add API key parameters
	queryParams.Set("_key", a.KeyID)
	queryParams.Set("_time", strconv.FormatInt(time.Now().Unix(), 10))
	queryParams.Set("_nonce", uuid.New().String())

	// Generate signature
	signature, err := a.generateSignature(method, path, queryParams, body)
	if err != nil {
		return err
	}

	// Add signature to query parameters as the last parameter
	// Note: Due to url.Values implementation, we can't control the order directly
	// We'll handle this in rest.go by ensuring _sign is the last parameter
	queryParams.Set("_sign", signature)
	return nil
}
