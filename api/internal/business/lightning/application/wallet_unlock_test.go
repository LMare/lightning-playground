package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Lmare/lightning-playground/api/internal/business/lightning/application"
	"github.com/Lmare/lightning-playground/api/internal/business/lightning/domain"
	"github.com/Lmare/lightning-playground/api/internal/testsupport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newUnlockService() (*application.NodeService, *testsupport.MockNodeInspector, *testsupport.MockWalletGateway) {
	inspector, wallet := &testsupport.MockNodeInspector{}, &testsupport.MockWalletGateway{}
	return application.NewNodeService(&testsupport.MockNodeRepository{}, inspector, wallet), inspector, wallet
}

func TestUnlockWallet_EmptyPassword(t *testing.T) {
	for _, pwd := range [][]byte{nil, {}} {
		svc, inspector, wallet := newUnlockService()

		err := svc.UnlockWallet(context.Background(), "lnd-0", pwd)

		assert.ErrorIs(t, err, domain.ErrWalletPasswordRequired)
		inspector.AssertNotCalled(t, "GetWalletState", mock.Anything, mock.Anything)
		wallet.AssertNotCalled(t, "Unlock", mock.Anything, mock.Anything, mock.Anything)
	}
}

func TestUnlockWallet_LockedCallsTheGatewayWithExactArguments(t *testing.T) {
	svc, inspector, wallet := newUnlockService()
	inspector.On("GetWalletState", mock.Anything, domain.NodeID("lnd-0")).Return(domain.WalletLocked, nil)
	boom := errors.New("gateway result")
	wallet.On("Unlock", mock.Anything, domain.NodeID("lnd-0"), []byte("s3cret")).Return(boom).Once()

	err := svc.UnlockWallet(context.Background(), "lnd-0", []byte("s3cret"))

	assert.Same(t, boom, err)
	wallet.AssertExpectations(t)

	wallet.On("Unlock", mock.Anything, domain.NodeID("lnd-0"), []byte("s3cret")).Return(nil).Once()
	assert.NoError(t, svc.UnlockWallet(context.Background(), "lnd-0", []byte("s3cret")))
}

func TestUnlockWallet_NotLockedStates(t *testing.T) {
	for _, st := range []domain.WalletState{domain.WalletServerActive, domain.WalletRPCActive, domain.WalletUnlocked, domain.WalletWaitingToStart} {
		svc, inspector, wallet := newUnlockService()
		inspector.On("GetWalletState", mock.Anything, mock.Anything).Return(st, nil)

		err := svc.UnlockWallet(context.Background(), "lnd-0", []byte("pwd"))

		assert.ErrorIs(t, err, domain.ErrWalletNotLocked, string(st))
		wallet.AssertNotCalled(t, "Unlock", mock.Anything, mock.Anything, mock.Anything)
	}
}

func TestUnlockWallet_NonExisting(t *testing.T) {
	svc, inspector, wallet := newUnlockService()
	inspector.On("GetWalletState", mock.Anything, mock.Anything).Return(domain.WalletNonExisting, nil)

	err := svc.UnlockWallet(context.Background(), "lnd-0", []byte("pwd"))

	assert.ErrorIs(t, err, domain.ErrWalletNotCreated)
	wallet.AssertNotCalled(t, "Unlock", mock.Anything, mock.Anything, mock.Anything)
}

func TestUnlockWallet_StateError(t *testing.T) {
	svc, inspector, wallet := newUnlockService()
	boom := errors.New("unreachable")
	inspector.On("GetWalletState", mock.Anything, mock.Anything).Return(domain.WalletState(""), boom)

	err := svc.UnlockWallet(context.Background(), "lnd-0", []byte("pwd"))

	assert.ErrorIs(t, err, boom)
	wallet.AssertNotCalled(t, "Unlock", mock.Anything, mock.Anything, mock.Anything)
}

func TestUnlockWallet_InvalidPasswordIsPropagated(t *testing.T) {
	svc, inspector, wallet := newUnlockService()
	inspector.On("GetWalletState", mock.Anything, mock.Anything).Return(domain.WalletLocked, nil)
	wallet.On("Unlock", mock.Anything, mock.Anything, mock.Anything).Return(domain.ErrInvalidWalletPassword)

	err := svc.UnlockWallet(context.Background(), "lnd-0", []byte("bad"))

	assert.ErrorIs(t, err, domain.ErrInvalidWalletPassword)
}
