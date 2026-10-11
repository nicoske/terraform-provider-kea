package kea

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

var _ Transport = &UnixTransport{}
var _ Transport = &HTTPTransport{}

type Transport interface {
	Send(ctx context.Context, req CommandRequest, resp *CommandResponse) error
}

type UnixTransport struct {
	SocketPath string
}

func (t *UnixTransport) Send(ctx context.Context, req CommandRequest, resp *CommandResponse) error {
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "unix", t.SocketPath)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	err = json.NewEncoder(conn).Encode(req)
	if err != nil {
		return err
	}

	if resp != nil {
		err = json.NewDecoder(conn).Decode(resp)
		if err != nil {
			return err
		}
	}
	return nil
}

type HTTPTransport struct {
	Endpoint string
	Client   *http.Client
	Username string
	Password string
}

func (t *HTTPTransport) Send(ctx context.Context, req CommandRequest, resp *CommandResponse) error {
	client := t.Client
	if client == nil {
		client = http.DefaultClient
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	hReq, err := http.NewRequestWithContext(ctx, "POST", t.Endpoint, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	hReq.Header.Set("Content-Type", "application/json")
	if t.Username != "" || t.Password != "" {
		hReq.SetBasicAuth(t.Username, t.Password)
	}

	hResp, err := client.Do(hReq)
	if err != nil {
		return err
	}
	defer func() { _ = hResp.Body.Close() }()

	if hResp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(hResp.Body, 1024))
		if msg := strings.TrimSpace(string(body)); msg != "" {
			return fmt.Errorf("http status %d: %s", hResp.StatusCode, msg)
		}
		return fmt.Errorf("http status: %d", hResp.StatusCode)
	}

	wrap := []*CommandResponse{resp}
	if resp != nil {
		err = json.NewDecoder(hResp.Body).Decode(&wrap)
		if err != nil {
			return err
		}
	}
	return nil
}
