package rest

import (
	"context"
	"errors"
)

type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Type         string `json:"token_type"`
	ClientID     string
	Expires      int `json:"expires_in"`
}

type tokenValue int

type withToken struct {
	context.Context
	token *Token
}

var (
	ErrNoClientID     = errors.New("no client_id has been provided for token renewal")
	ErrNoRefreshToken = errors.New("no refresh token is available and access token has expired")
)

func (w *withToken) Value(v any) any {
	if _, ok := v.(tokenValue); ok {
		return w.token
	}

	return w.Context.Value(v)
}

func (t *Token) Use(ctx context.Context) context.Context {
	return &withToken{ctx, t}
}

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
