package lnd_test

import (
	"context"
	"errors"
	"testing"

	lnrpc "github.com/Lmare/lightning-playground/backend/gRPC/github.com/lightningnetwork/lnd/lnrpc"
	"github.com/Lmare/lightning-playground/backend/internal/business/lightning/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func amt(sat uint64) *lnrpc.Amount { return &lnrpc.Amount{Sat: sat, Msat: sat * 1000} }

func TestGetBalance_MapsEveryField(t *testing.T) {
	gw, factory := newGateway()
	factory.Lightning.On("WalletBalance", mock.Anything, mock.Anything).Return(&lnrpc.WalletBalanceResponse{
		ConfirmedBalance: 1000, UnconfirmedBalance: 250, LockedBalance: 11, ReservedBalanceAnchorChan: 22,
	}, nil)
	factory.Lightning.On("ChannelBalance", mock.Anything, mock.Anything).Return(&lnrpc.ChannelBalanceResponse{
		LocalBalance: amt(600), RemoteBalance: amt(390),
		UnsettledLocalBalance: amt(10), UnsettledRemoteBalance: amt(3),
		PendingOpenLocalBalance: amt(50), PendingOpenRemoteBalance: amt(7),
	}, nil)

	got, err := gw.GetBalance(context.Background(), "lnd-0")

	assert.NoError(t, err)
	assert.Equal(t, domain.Balance{
		OnChain: domain.OnChainBalance{ConfirmedSat: 1000, UnconfirmedSat: 250, LockedSat: 11, ReservedAnchorSat: 22},
		Channels: domain.ChannelBalance{
			LocalSat: 600, RemoteSat: 390, PendingOpenLocalSat: 50, PendingOpenRemoteSat: 7,
			UnsettledLocalSat: 10, UnsettledRemoteSat: 3,
		},
	}, got)
	assert.Equal(t, "lnd-0.lnd.test:10009", factory.Connections[0].Address)
}

func TestGetBalance_NilAmountsAreZero(t *testing.T) {
	gw, factory := newGateway()
	factory.Lightning.On("WalletBalance", mock.Anything, mock.Anything).Return(&lnrpc.WalletBalanceResponse{ConfirmedBalance: 5}, nil)
	factory.Lightning.On("ChannelBalance", mock.Anything, mock.Anything).Return(&lnrpc.ChannelBalanceResponse{LocalBalance: amt(9)}, nil)

	got, err := gw.GetBalance(context.Background(), "lnd-0")

	assert.NoError(t, err)
	assert.Equal(t, int64(5), got.OnChain.ConfirmedSat)
	assert.Equal(t, int64(9), got.Channels.LocalSat)
	assert.Equal(t, int64(0), got.Channels.PendingOpenLocalSat)
	assert.Equal(t, int64(0), got.Channels.RemoteSat)
}

func TestGetBalance_WalletBalanceError(t *testing.T) {
	gw, factory := newGateway()
	boom := errors.New("wallet balance failed")
	factory.Lightning.On("WalletBalance", mock.Anything, mock.Anything).Return(nil, boom)

	_, err := gw.GetBalance(context.Background(), "lnd-0")

	assert.ErrorIs(t, err, boom)
	factory.Lightning.AssertNotCalled(t, "ChannelBalance", mock.Anything, mock.Anything)
}

func TestGetBalance_ChannelBalanceError(t *testing.T) {
	gw, factory := newGateway()
	boom := errors.New("channel balance failed")
	factory.Lightning.On("WalletBalance", mock.Anything, mock.Anything).Return(&lnrpc.WalletBalanceResponse{}, nil)
	factory.Lightning.On("ChannelBalance", mock.Anything, mock.Anything).Return(nil, boom)

	_, err := gw.GetBalance(context.Background(), "lnd-0")

	assert.ErrorIs(t, err, boom)
}

func TestGetBalance_FactoryError(t *testing.T) {
	gw, factory := newGateway()
	factory.Err = errors.New("cannot connect")

	_, err := gw.GetBalance(context.Background(), "lnd-0")

	assert.Error(t, err)
	factory.Lightning.AssertNotCalled(t, "WalletBalance", mock.Anything, mock.Anything)
}
