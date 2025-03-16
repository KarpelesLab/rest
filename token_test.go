package rest

import (
	"context"
	"testing"
)

// TestTokenContext tests token context manipulation functions
func TestTokenContext(t *testing.T) {
	// Create a test token
	token := &Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		Type:         "Bearer",
		ClientID:     "test-client-id",
		Expires:      3600,
	}

	// Test Use method
	baseCtx := context.Background()
	tokenCtx := token.Use(baseCtx)

	// Verify the context contains the token
	retrievedToken, ok := tokenCtx.Value(tokenValue(0)).(*Token)
	if !ok {
		t.Errorf("Failed to retrieve token from context")
	}

	if retrievedToken != token {
		t.Errorf("Retrieved token = %v, want %v", retrievedToken, token)
	}

	// Test Value method of withToken
	if wt, ok := tokenCtx.(*withToken); ok {
		// Test retrieving the token
		result := wt.Value(tokenValue(0))
		if result != token {
			t.Errorf("withToken.Value(tokenValue) = %v, want %v", result, token)
		}

		// Test retrieving something else, should delegate to parent context
		type keyType string
		key := keyType("test-key")
		result = wt.Value(key)
		if result != nil {
			t.Errorf("withToken.Value(other) = %v, want nil", result)
		}
	} else {
		t.Errorf("tokenCtx is not a *withToken, got %T", tokenCtx)
	}
}

// TestTokenRenew is a limited test for the token renew functionality
// It can't actually make API calls, so we just test error handling
func TestTokenRenew(t *testing.T) {
	// Test with missing client ID
	token1 := &Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		Type:         "Bearer",
		ClientID:     "", // Missing client ID should cause ErrNoClientID
		Expires:      3600,
	}

	err := token1.renew(context.Background())
	if err != ErrNoClientID {
		t.Errorf("token.renew() with missing ClientID = %v, want %v", err, ErrNoClientID)
	}

	// Test with missing refresh token
	token2 := &Token{
		AccessToken:  "test-access-token",
		RefreshToken: "", // Missing refresh token should cause ErrNoRefreshToken
		Type:         "Bearer",
		ClientID:     "test-client-id",
		Expires:      3600,
	}

	err = token2.renew(context.Background())
	if err != ErrNoRefreshToken {
		t.Errorf("token.renew() with missing RefreshToken = %v, want %v", err, ErrNoRefreshToken)
	}
}