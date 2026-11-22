package application

import (
	"context"

	"github.com/Lmare/lightning-playground/backend/internal/business/lightning/domain"
	"github.com/Lmare/lightning-playground/backend/internal/business/lightning/port"
)

func NewChannelService(channels port.ChannelGateway, payments port.PaymentGateway, notifier port.PaymentNotifier) *ChannelService {
	return &ChannelService{channels: channels, payments: payments, notifier: notifier}
}

// ChannelService is the use case set about peers, channels, invoices and payments.
type ChannelService struct {
	channels port.ChannelGateway
	payments port.PaymentGateway
	notifier port.PaymentNotifier
}

// AddPeer connects the node to the peer described by a "<pubkey>@<host>" URI.
func (s *ChannelService) AddPeer(ctx context.Context, id domain.NodeID, uri string) error {
	peer, err := domain.ParsePeerAddress(uri)
	if err != nil {
		return err
	}
	return s.channels.AddPeer(ctx, id, peer)
}

// OpenChannel opens a channel of amountSat with the peer.
func (s *ChannelService) OpenChannel(ctx context.Context, id domain.NodeID, peerPubKeyHex string, amountSat int64) error {
	if amountSat <= 0 {
		return domain.ErrInvalidAmount
	}
	return s.channels.OpenChannel(ctx, id, peerPubKeyHex, amountSat)
}

// CreateInvoice returns the payment request of a new invoice.
func (s *ChannelService) CreateInvoice(ctx context.Context, id domain.NodeID, memo string, amountSat int64) (string, error) {
	if amountSat <= 0 {
		return "", domain.ErrInvalidAmount
	}
	return s.payments.CreateInvoice(ctx, id, memo, amountSat)
}

// Pay starts the payment of an invoice; its progress is reported asynchronously to the notifier.
func (s *ChannelService) Pay(ctx context.Context, id domain.NodeID, paymentRequest string) error {
	progress, err := s.payments.PayInvoice(ctx, id, paymentRequest)
	if err != nil {
		return err
	}
	s.notifier.Watch(progress)
	return nil
}
