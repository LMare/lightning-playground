package port

import (
	"context"

	"github.com/Lmare/lightning-playground/backend/internal/business/lightning/domain"
)

// NodeInspector is the outbound port to read and configure a Lightning node.
type NodeInspector interface {
	GetSummary(ctx context.Context, id domain.NodeID) (domain.NodeSummary, error)
	// GetWalletState returns the wallet state. An error means the node could not be queried.
	GetWalletState(ctx context.Context, id domain.NodeID) (domain.WalletState, error)
	GetInfo(ctx context.Context, id domain.NodeID) (domain.NodeInfo, error)
	UpdateAppearance(ctx context.Context, id domain.NodeID, alias, color string) error
}

// ChannelGateway is the outbound port to connect a node to the network.
type ChannelGateway interface {
	AddPeer(ctx context.Context, id domain.NodeID, peer domain.PeerAddress) error
	OpenChannel(ctx context.Context, id domain.NodeID, peerPubKeyHex string, amountSat int64) error
}

// PaymentGateway is the outbound port to receive and send payments.
type PaymentGateway interface {
	// CreateInvoice returns the payment request of a short-lived invoice.
	CreateInvoice(ctx context.Context, id domain.NodeID, memo string, amountSat int64) (string, error)
	// PayInvoice starts the payment and returns its progress. The caller must Close it.
	PayInvoice(ctx context.Context, id domain.NodeID, paymentRequest string) (PaymentProgress, error)
}

// PaymentProgress yields the updates of a running payment; Recv returns io.EOF at the end.
type PaymentProgress interface {
	Recv() (domain.PaymentUpdate, error)
	Close() error
}

// PaymentNotifier is the outbound port used to tell the user about a payment progress.
type PaymentNotifier interface {
	// Watch consumes the progress asynchronously and closes it when finished.
	Watch(progress PaymentProgress)
}
