package rest

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/KarpelesLab/pjson"
)

// SpotClient is an interface fullfilled by spotlib.Client that contains
// everything we really care about, and helps avoid dependency loops
type SpotClient interface {
	Query(ctx context.Context, target string, body []byte) ([]byte, error)
}

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
