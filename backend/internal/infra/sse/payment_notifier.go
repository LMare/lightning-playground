// Package sse hosts the SSE hub and its adapter to the lightning notification ports.
package sse

import (
	"github.com/Lmare/lightning-playground/backend/internal/business/lightning/domain"
	"github.com/Lmare/lightning-playground/backend/internal/business/lightning/port"
)

// PaymentNotifier broadcasts payment updates to every SSE client.
type PaymentNotifier struct{}

func NewPaymentNotifier() *PaymentNotifier { return &PaymentNotifier{} }

var _ port.PaymentNotifier = (*PaymentNotifier)(nil)

func (n *PaymentNotifier) Watch(progress port.PaymentProgress) {
	StreamResult(StreamWrapper[domain.PaymentUpdate]{
		RecvCallback: func() (*domain.PaymentUpdate, error) {
			update, err := progress.Recv()
			if err != nil {
				progress.Close() // the stream is over (EOF or error)
				return nil, err
			}
			return &update, nil
		},
		CloseCallback: progress.Close,
	})
}
