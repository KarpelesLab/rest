package rest

import (
	"context"
	"crypto/ed25519"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// TestNewApiKey verifies the API key loading function.
func TestNewApiKey(t *testing.T) {
	// Read the API key file to use a real key for testing
	data, err := os.ReadFile("api.txt")
	if err != nil {
		t.Skipf("Skipping test, could not read API key file: %v", err)
		return
	}

	// Parse the key and secret from the file
	parts := strings.SplitN(strings.TrimSpace(string(data)), ":", 2)
	if len(parts) != 2 {
		t.Skipf("Skipping test, API key file has invalid format")
		return
	}

	testKeyID := parts[0]
	testSecret := parts[1]
	
	// Test with the real key
	apiKey, err := NewApiKey(testKeyID, testSecret)
	if err != nil {
		t.Fatalf("Failed to load API key: %v", err)
	}
	
	// Verify the key ID was properly loaded
	if apiKey.KeyID != testKeyID {
		t.Errorf("Expected key ID to be %s, got %s", testKeyID, apiKey.KeyID)
	}
	
	// Verify the secret was properly loaded
	if len(apiKey.SecretKey) != ed25519.PrivateKeySize {
		t.Errorf("Expected secret key length to be %d, got %d", ed25519.PrivateKeySize, len(apiKey.SecretKey))
	}
	
	// Test with an invalid base64 string - should return error
	_, err = NewApiKey("test-key-id", "invalid-base64-!@#$")
	if err == nil {
		t.Errorf("Expected error for invalid base64, got nil")
	}
}

// TestApiKey verifies the context functionality of the API key implementation.
func TestApiKey(t *testing.T) {
	// Read the API key file to use a real key for testing
	data, err := os.ReadFile("api.txt")
	if err != nil {
		t.Skipf("Skipping test, could not read API key file: %v", err)
		return
	}

	// Parse the key and secret from the file
	parts := strings.SplitN(strings.TrimSpace(string(data)), ":", 2)
	if len(parts) != 2 {
		t.Skipf("Skipping test, API key file has invalid format")
		return
	}

	testKeyID := parts[0]
	testSecret := parts[1]
	
	// Load a test API key
	apiKey, err := NewApiKey(testKeyID, testSecret)
	if err != nil {
		t.Fatalf("Failed to load API key: %v", err)
	}

	// Test context passing
	ctx := context.Background()
	ctx = apiKey.Use(ctx)

	// Verify the context contains the API key
	retrievedKey, ok := ctx.Value(apiKeyValue(0)).(*ApiKey)
	if !ok || retrievedKey == nil {
		t.Fatal("Expected context to contain API key")
	}

	if retrievedKey.KeyID != apiKey.KeyID {
		t.Errorf("Expected retrieved key ID to be %s, got %s", apiKey.KeyID, retrievedKey.KeyID)
	}
}

// TestReadApiFromFile tests loading an API key from a file.
// Note: For testing purposes only, we use a simple file format "key:secret" where secret is 
// already encoded in base64url format (using - and _ characters instead of + and /).
func TestReadApiFromFile(t *testing.T) {
	// Read the API key file
	data, err := os.ReadFile("api.txt")
	if err != nil {
		t.Skipf("Skipping test, could not read API key file: %v", err)
		return
	}

	// Parse the key and secret - for TESTING PURPOSES ONLY
	// In real code, the key and secret would be provided by the API service
	parts := strings.SplitN(strings.TrimSpace(string(data)), ":", 2)
	if len(parts) != 2 {
		t.Skipf("Skipping test, API key file has invalid format")
		return
	}

	keyID := parts[0]
	secret := parts[1]  // Secret is already base64-encoded in the file

	// Load the API key
	apiKey, err := NewApiKey(keyID, secret)
	if err != nil {
		t.Fatalf("Failed to load API key: %v", err)
	}

	// Check the key ID
	if apiKey.KeyID != keyID {
		t.Errorf("Expected key ID to be %s, got %s", keyID, apiKey.KeyID)
	}

	// Verify the secret was properly loaded
	if len(apiKey.SecretKey) != ed25519.PrivateKeySize {
		t.Errorf("Expected secret key length to be %d, got %d", ed25519.PrivateKeySize, len(apiKey.SecretKey))
	}
}

// TestBodyHash verifies the body hashing mechanism.
func TestBodyHash(t *testing.T) {
	// Test empty body hash
	emptyHash := BodyHash(nil)
	expectedEmptyHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if emptyHash != expectedEmptyHash {
		t.Errorf("Expected empty hash to be %s, got %s", expectedEmptyHash, emptyHash)
	}

	// Test with content
	content := []byte("test content")
	contentHash := BodyHash(content)
	expectedContentHash := "6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72"
	if contentHash != expectedContentHash {
		t.Errorf("Expected content hash to be %s, got %s", expectedContentHash, contentHash)
	}
}

// TestUserGetWithApiKey tests a real API call using the API key.
// This ensures the integration works as expected with actual requests.
func TestUserGetWithApiKey(t *testing.T) {
	// Read the API key file
	data, err := os.ReadFile("api.txt")
	if err != nil {
		t.Skipf("Skipping test, could not read API key file: %v", err)
		return
	}

	// Parse the key and secret - for TESTING PURPOSES ONLY
	// In real code, the key and secret would be provided by the API service
	parts := strings.SplitN(strings.TrimSpace(string(data)), ":", 2)
	if len(parts) != 2 {
		t.Skipf("Skipping test, API key file has invalid format")
		return
	}

	keyID := parts[0]
	secret := parts[1]  // Secret is already base64-encoded in the file

	// Load the API key
	apiKey, err := NewApiKey(keyID, secret)
	if err != nil {
		t.Fatalf("Failed to load API key: %v", err)
	}

	// Create a context with the API key
	ctx := context.Background()
	ctx = apiKey.Use(ctx)

	// Try to get user info
	type UserInfo struct {
		UserID string `json:"User__"`
	}

	// Skip the test if we can't reach the server
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	_, err = client.Get("https://" + Host)
	if err != nil {
		t.Skipf("Skipping test, cannot reach server %s: %v", Host, err)
		return
	}
	
	// Print which server we're connecting to for debugging
	t.Logf("Testing User:get API call with host: %s", Host)

	var result UserInfo
	err = Apply(ctx, "User:get", "GET", nil, &result)
	
	// This test should fail if the API key doesn't work
	if err != nil {
		t.Fatalf("User:get request returned error: %v", err)
		return
	}
	
	t.Logf("User:get request succeeded, got user ID: %s", result.UserID)
	
	// Verify that we got a valid user ID
	if result.UserID == "" {
		t.Errorf("User:get request succeeded but returned empty UserID")
	}
	
	// Verify the UserID field is in the expected format (should start with usr-)
	if len(result.UserID) < 5 || result.UserID[:4] != "usr-" {
		t.Errorf("UserID has invalid format: %q", result.UserID)
	}
	
	// Log the user ID for debugging
	t.Logf("Got UserID: %s", result.UserID)
}

// Example provides a documented example of how API key authentication works.
// This is not a real test but serves as documentation for users.
func Example() {
	// API key authentication is automatically handled by the library
	// You obtain your API key (key ID and secret) from the service provider
	// 
	// 1. Load your API key into your application
	// 2. Create a context with your API key
	// ctx := context.Background()
	// ctx = apiKey.Use(ctx)
	// 
	// 3. Use the context with any API call
	// result, err := Do(ctx, "User:get", "GET", nil)
	// 
	// The library automatically:
	// - Adds the key ID to the request parameters
	// - Adds a timestamp and nonce for security
	// - Calculates a cryptographic signature
	// - Ensures the signature is the last parameter in the URL
	//
	// All of this happens behind the scenes when you use the API key context
	// with rest.Do(), rest.Apply(), or rest.As() functions.
}