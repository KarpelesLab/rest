package rest

import (
	"context"
	"errors"
)

// Token represents an OAuth2 token with refresh capabilities.
// It contains both access and refresh tokens and methods to use them in requests.
type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Type         string `json:"token_type"`
	ClientID     string
	Expires      int `json:"expires_in"`
}

// tokenValue is a type used as a context key for token storage.
type tokenValue int

// withToken is a context wrapper that holds a token value.
type withToken struct {
	context.Context
	token *Token
}

var (
	// ErrNoClientID is returned when token renewal is attempted without a client ID
	ErrNoClientID     = errors.New("no client_id has been provided for token renewal")
	// ErrNoRefreshToken is returned when token renewal is attempted without a refresh token
	ErrNoRefreshToken = errors.New("no refresh token is available and access token has expired")
)

// Value implements the context.Context Value method for withToken.
// It returns the token for tokenValue keys and delegates to the parent context otherwise.
func (w *withToken) Value(v any) any {
	if _, ok := v.(tokenValue); ok {
		return w.token
	}

	return w.Context.Value(v)
}

// Use returns a new context that includes this token for authentication.
// The token will be used for all REST API calls that use this context.
func (t *Token) Use(ctx context.Context) context.Context {
	return &withToken{ctx, t}
}

// renew attempts to refresh an expired access token using the refresh token.
// It makes a request to the OAuth2:token endpoint with the refresh token.
func (t *Token) renew(ctx context.Context) error {
	// perform renew of token via OAuth2:token endpoint
	ctx = &withToken{ctx, nil} // set token to nil

	if t.ClientID == "" {
		return ErrNoClientID
	}
	if t.RefreshToken == "" {
		return ErrNoRefreshToken
	}

	req := map[string]any{
		"grant_type":    "refresh_token",
		"client_id":     t.ClientID,
		"refresh_token": t.RefreshToken,
		"noraw":         true,
	}

	err := Apply(ctx, "OAuth2:token", "POST", req, t)
	if err != nil {
		return err
	}

	// Apply will have updated AccessToken
	return nil
}
