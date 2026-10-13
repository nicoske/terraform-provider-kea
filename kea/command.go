package kea

import (
	"context"
	"encoding/json"
	"fmt"
)

type ResultCode int

const (
	ResultSuccess     ResultCode = 0
	ResultError       ResultCode = 1
	ResultUnsupported ResultCode = 2
	ResultEmpty       ResultCode = 3
	ResultConflict    ResultCode = 4
)

type CommandRequest struct {
	Command   string          `json:"command"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type CommandResponse struct {
	Result    ResultCode      `json:"result"`
	Text      string          `json:"text"`
	Arguments json.RawMessage `json:"arguments"`
}

func ExecWithResponse[T any](ctx context.Context, trans Transport, command string, arguments any) (*T, error) {
	var result T

	req, err := newCommand(command, arguments)
	if err != nil {
		return nil, err
	}

	var resp CommandResponse
	err = roundTrip(ctx, trans, req, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Result == ResultEmpty {
		return nil, nil
	}

	err = json.Unmarshal(resp.Arguments, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func Exec(ctx context.Context, trans Transport, command string, arguments any) error {
	req, err := newCommand(command, arguments)
	if err != nil {
		return err
	}
	var resp CommandResponse
	return roundTrip(ctx, trans, req, &resp)
}

func newCommand(command string, arguments any) (CommandRequest, error) {
	var req CommandRequest

	encArgs, err := json.Marshal(arguments)
	if err != nil {
		return req, err
	}

	req = CommandRequest{
		Command:   command,
		Arguments: json.RawMessage(encArgs),
	}
	return req, nil
}

func roundTrip(ctx context.Context, trans Transport, req CommandRequest, resp *CommandResponse) error {
	err := trans.Send(ctx, req, resp)
	if err != nil {
		return err
	}

	switch resp.Result {
	case ResultSuccess:
		return nil
	case ResultEmpty:
		return nil
	case ResultError:
		return fmt.Errorf("general error: %s", resp.Text)
	case ResultUnsupported:
		return fmt.Errorf("unsupported command: %s", resp.Text)
	case ResultConflict:
		return fmt.Errorf("conflict: %s", resp.Text)
	default:
		return fmt.Errorf("unrecognized result code: %d", resp.Result)
	}
}
