package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Lmare/lightning-playground/backend/internal/business/lightning/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var sampleBalance = domain.Balance{
	OnChain:  domain.OnChainBalance{ConfirmedSat: 1000, UnconfirmedSat: 250, LockedSat: 1, ReservedAnchorSat: 2},
	Channels: domain.ChannelBalance{LocalSat: 600, RemoteSat: 390, PendingOpenLocalSat: 50, UnsettledLocalSat: 10},
}

func TestGetBalance_ReadyStatesDelegate(t *testing.T) {
	for _, st := range []domain.WalletState{domain.WalletServerActive, domain.WalletRPCActive} {
		svc, inspector, wallet := newUnlockService()
		inspector.On("GetWalletState", mock.Anything, domain.NodeID("lnd-0")).Return(st, nil)
		wallet.On("GetBalance", mock.Anything, domain.NodeID("lnd-0")).Return(sampleBalance, nil).Once()

		got, err := svc.GetBalance(context.Background(), "lnd-0")

		assert.NoError(t, err, string(st))
		assert.Equal(t, sampleBalance, got)
		wallet.AssertExpectations(t)
	}
}

func TestGetBalance_NotReadyStates(t *testing.T) {
	for _, st := range []domain.WalletState{domain.WalletLocked, domain.WalletNonExisting, domain.WalletUnlocked, domain.WalletWaitingToStart, domain.WalletUnknown} {
		svc, inspector, wallet := newUnlockService()
		inspector.On("GetWalletState", mock.Anything, mock.Anything).Return(st, nil)

		_, err := svc.GetBalance(context.Background(), "lnd-0")

		assert.ErrorIs(t, err, domain.ErrWalletNotReady, string(st))
		wallet.AssertNotCalled(t, "GetBalance", mock.Anything, mock.Anything)
	}
}

func TestGetBalance_StateError(t *testing.T) {
	svc, inspector, wallet := newUnlockService()
	boom := errors.New("unreachable")
	inspector.On("GetWalletState", mock.Anything, mock.Anything).Return(domain.WalletState(""), boom)

	_, err := svc.GetBalance(context.Background(), "lnd-0")

	assert.ErrorIs(t, err, boom)
	assert.NotErrorIs(t, err, domain.ErrWalletNotReady)
	wallet.AssertNotCalled(t, "GetBalance", mock.Anything, mock.Anything)
}

func TestGetBalance_GatewayError(t *testing.T) {
	svc, inspector, wallet := newUnlockService()
	boom := errors.New("lnd down")
	inspector.On("GetWalletState", mock.Anything, mock.Anything).Return(domain.WalletServerActive, nil)
	wallet.On("GetBalance", mock.Anything, mock.Anything).Return(domain.Balance{}, boom)

	_, err := svc.GetBalance(context.Background(), "lnd-0")

	assert.ErrorIs(t, err, boom)
}
