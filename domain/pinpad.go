package domain

import (
	"context"
	"errors"
)

var ErrEDCNotConnected = errors.New("edc is not connected")

type PinChangeRequest struct {
	TerminalID string `json:"terminal_id"`
	TraceNo    string `json:"trace_no"`
}

type PinpadResponse struct {
	TerminalID  string `json:"terminal_id"`
	TraceNo     string `json:"trace_no"`
	Status      string `json:"status"`
	EncPinBlock string `json:"enc_pin_block,omitempty"`
}

type PinpadUsecase interface {
	ProcessPinChange(ctx context.Context, req PinChangeRequest) (*PinpadResponse, error)
}

type EDCGateway interface {
	SendAndAwaitResponse(ctx context.Context, req PinChangeRequest) (*PinpadResponse, error)
}
