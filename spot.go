package rest

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/KarpelesLab/pjson"
)

// SpotClient is an interface fulfilled by spotlib.Client that provides
// the necessary functionality for making API requests through a Spot connection.
// Using this interface helps avoid dependency loops between packages.
type SpotClient interface {
	Query(ctx context.Context, target string, body []byte) ([]byte, error)
}

// SpotApply makes a REST API request through a SpotClient and unmarshals the response into target.
// This is similar to Apply but uses a SpotClient for the request.
//
// Parameters:
// - ctx: Context for the request
// - client: SpotClient to use for the API request
// - path: API endpoint path
// - method: HTTP method (GET, POST, PUT, etc.)
// - param: Request parameters or body content
// - target: Destination object for unmarshaled response data
//
// Returns an error if the request fails or response cannot be unmarshaled.
func SpotApply(ctx context.Context, client SpotClient, path, method string, param any, target any) error {
	res, err := SpotDo(ctx, client, path, method, param)
	if err != nil {
		return err
	}
	err = pjson.UnmarshalContext(ctx, res.Data, target)
	if Debug && err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("failed to parse json: %s\n%s", err, res.Data), "event", "rest:not_json")
	}
	return err
}

// SpotAs makes a REST API request through a SpotClient and returns the response data unmarshaled into type T.
// This is a generic version of SpotApply that returns the target object directly.
//
// Parameters:
// - ctx: Context for the request
// - client: SpotClient to use for the API request
// - path: API endpoint path
// - method: HTTP method (GET, POST, PUT, etc.)
// - param: Request parameters or body content
//
// Returns the unmarshaled object of type T and any error encountered.
func SpotAs[T any](ctx context.Context, client SpotClient, path, method string, param any) (T, error) {
	var target T
	res, err := SpotDo(ctx, client, path, method, param)
	if err != nil {
		return target, err
	}
	err = pjson.UnmarshalContext(ctx, res.Data, &target)
	if Debug && err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("failed to parse json: %s\n%s", err, res.Data), "event", "rest:not_json")
	}
	return target, err
}

// SpotDo executes a REST API request through a SpotClient and returns the raw Response object.
// This is the base function used by SpotApply and SpotAs.
//
// Parameters:
// - ctx: Context for the request
// - client: SpotClient to use for the API request
// - path: API endpoint path
// - method: HTTP method (GET, POST, PUT, etc.)
// - param: Request parameters or body content
//
// Returns the raw Response object and any error encountered during the request.
func SpotDo(ctx context.Context, client SpotClient, path, method string, param any) (*Response, error) {
	req := map[string]any{
		"path":   path,
		"verb":   method,
		"params": param,
	}
	buf, err := pjson.Marshal(req)
	if err != nil {
		return nil, err
	}
	respbuf, err := client.Query(ctx, "@/p_api", buf)
	if err != nil {
		return nil, err
	}

	var resp *Response
	err = pjson.Unmarshal(respbuf, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
