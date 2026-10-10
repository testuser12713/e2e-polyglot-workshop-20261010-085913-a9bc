// Package queue publishes workshop events to the Valkey stream consumed by the
// invoice worker.
package queue

import (
	"context"
	"errors"

	valkey "github.com/valkey-io/valkey-go"
)

// Stream entry fields of an invoice message. The payload is
// {order_number, plate}.
const (
	fieldOrderNumber = "order_number"
	fieldPlate       = "plate"
)

// InvoiceStream is the Valkey stream the invoice worker consumes.
const InvoiceStream = "workshop:invoices"

// ErrNotImplemented is returned by the seams the feature tickets fill in.
var ErrNotImplemented = errors.New("not_implemented")

// Publisher writes messages to the invoice stream.
type Publisher struct {
	client valkey.Client
}

// NewPublisher connects a publisher from a Valkey URL (redis://...). The
// connection is lazy: no command is sent until PublishInvoice is called.
func NewPublisher(url string) (*Publisher, error) {
	opt, err := valkey.ParseURL(url)
	if err != nil {
		return nil, err
	}
	client, err := valkey.NewClient(opt)
	if err != nil {
		return nil, err
	}
	return &Publisher{client: client}, nil
}

// Close releases the underlying client.
func (p *Publisher) Close() {
	if p.client != nil {
		p.client.Close()
	}
}

// PublishInvoice appends one message with the order number and plate to the
// invoice stream (AC-07). It returns an error when Valkey rejects the command
// so the caller can roll the status transition back.
func (p *Publisher) PublishInvoice(ctx context.Context, orderNumber, plate string) error {
	if p == nil || p.client == nil {
		return errors.New("invoice publisher is not configured")
	}
	cmd := p.client.B().Xadd().
		Key(InvoiceStream).
		Id("*").
		FieldValue().
		FieldValue(fieldOrderNumber, orderNumber).
		FieldValue(fieldPlate, plate).
		Build()
	return p.client.Do(ctx, cmd).Error()
}
