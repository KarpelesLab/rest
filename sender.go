package rest

import (
	"context"
	"io"
)

type SenderInterface interface {
	Send(from string, to []string, msg io.WriterTo) error
}

type TokenSender struct {
	Token string
}

type restSender struct{}

var Sender SenderInterface = restSender{}

func (s restSender) Send(from string, to []string, msg io.WriterTo) error {
	reader, writer := io.Pipe()
	defer reader.Close()
	go func() {
		defer writer.Close()
		msg.WriteTo(writer)
	}()
	_, err := Upload(context.Background(), "MTA:send", "POST", map[string]any{"from": from, "to": to}, reader, "message/rfc822")
	return err
}

func (s *TokenSender) Send(from string, to []string, msg io.WriterTo) error {
	reader, writer := io.Pipe()
	defer reader.Close()
	go func() {
		defer writer.Close()
		msg.WriteTo(writer)
	}()
	_, err := Upload(context.Background(), "MTA:send", "POST", map[string]any{"from": from, "to": to, "token": s.Token}, reader, "message/rfc822")
	return err
}
