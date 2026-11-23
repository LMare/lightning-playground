package testsupport

import (
	"context"

	"github.com/Lmare/lightning-playground/backend/internal/business/lightning/domain"
	"github.com/Lmare/lightning-playground/backend/internal/business/lightning/port"
	"github.com/stretchr/testify/mock"
)

// MockNodeInspector mocks port.NodeInspector.
type MockNodeInspector struct{ mock.Mock }

func (m *MockNodeInspector) GetSummary(ctx context.Context, id domain.NodeID) (domain.NodeSummary, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.NodeSummary), args.Error(1)
}

func (m *MockNodeInspector) GetWalletState(ctx context.Context, id domain.NodeID) (domain.WalletState, error) {
	args := m.Called(ctx, id)
	state, _ := args.Get(0).(domain.WalletState)
	return state, args.Error(1)
}

func (m *MockNodeInspector) GetInfo(ctx context.Context, id domain.NodeID) (domain.NodeInfo, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.NodeInfo), args.Error(1)
}

func (m *MockNodeInspector) UpdateAppearance(ctx context.Context, id domain.NodeID, alias, color string) error {
	return m.Called(ctx, id, alias, color).Error(0)
}

// MockChannelGateway mocks port.ChannelGateway.
type MockChannelGateway struct{ mock.Mock }

func (m *MockChannelGateway) AddPeer(ctx context.Context, id domain.NodeID, peer domain.PeerAddress) error {
	return m.Called(ctx, id, peer).Error(0)
}

func (m *MockChannelGateway) OpenChannel(ctx context.Context, id domain.NodeID, pubKey string, amount int64) error {
	return m.Called(ctx, id, pubKey, amount).Error(0)
}

// MockPaymentGateway mocks port.PaymentGateway.
type MockPaymentGateway struct{ mock.Mock }

func (m *MockPaymentGateway) CreateInvoice(ctx context.Context, id domain.NodeID, memo string, amount int64) (string, error) {
	args := m.Called(ctx, id, memo, amount)
	return args.String(0), args.Error(1)
}

func (m *MockPaymentGateway) PayInvoice(ctx context.Context, id domain.NodeID, paymentRequest string) (port.PaymentProgress, error) {
	args := m.Called(ctx, id, paymentRequest)
	progress, _ := args.Get(0).(port.PaymentProgress)
	return progress, args.Error(1)
}

// MockWalletGateway mocks port.WalletGateway.
type MockWalletGateway struct{ mock.Mock }

func (m *MockWalletGateway) Unlock(ctx context.Context, id domain.NodeID, password []byte) error {
	return m.Called(ctx, id, password).Error(0)
}

// MockPaymentNotifier mocks port.PaymentNotifier.
type MockPaymentNotifier struct{ mock.Mock }

func (m *MockPaymentNotifier) Watch(progress port.PaymentProgress) { m.Called(progress) }

var (
	_ port.NodeInspector   = (*MockNodeInspector)(nil)
	_ port.ChannelGateway  = (*MockChannelGateway)(nil)
	_ port.PaymentGateway  = (*MockPaymentGateway)(nil)
	_ port.PaymentNotifier = (*MockPaymentNotifier)(nil)
	_ port.WalletGateway   = (*MockWalletGateway)(nil)
)
