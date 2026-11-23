package lnd_test

import (
	"context"
	"errors"
	"testing"

	lnrpc "github.com/Lmare/lightning-playground/backend/gRPC/github.com/lightningnetwork/lnd/lnrpc"
	"github.com/Lmare/lightning-playground/backend/internal/business/lightning/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const unlockPassword = "Sup3r-S3cret-Pwd"

var sentinels = []error{domain.ErrInvalidWalletPassword, domain.ErrWalletNotLocked, domain.ErrWalletNotCreated, domain.ErrWalletPasswordRequired}

func TestUnlock_Success(t *testing.T) {
	gw, factory := newGateway()
	factory.WalletUnlocker.On("UnlockWallet", mock.Anything, mock.MatchedBy(func(r *lnrpc.UnlockWalletRequest) bool {
		return string(r.WalletPassword) == unlockPassword
	})).Return(&lnrpc.UnlockWalletResponse{}, nil)

	err := gw.Unlock(context.Background(), "lnd-0", []byte(unlockPassword))

	assert.NoError(t, err)
	factory.WalletUnlocker.AssertNumberOfCalls(t, "UnlockWallet", 1)
	assert.Equal(t, "lnd-0.lnd.test:10009", factory.Connections[0].Address)
}

func TestUnlock_ErrorMapping(t *testing.T) {
	cases := []struct {
		name     string
		grpcErr  error
		sentinel error
	}{
		{"invalid passphrase", status.Error(codes.Unknown, "invalid passphrase for master public key"), domain.ErrInvalidWalletPassword},
		{"already unlocked", status.Error(codes.Unknown, "wallet already unlocked, WalletUnlocker service is no longer available"), domain.ErrWalletNotLocked},
		{"wallet not found", status.Error(codes.Unknown, "wallet not found"), domain.ErrWalletNotCreated},
		{"other grpc error", status.Error(codes.Unavailable, "connection refused"), nil},
		{"plain error", errors.New("boom"), nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gw, factory := newGateway()
			factory.WalletUnlocker.On("UnlockWallet", mock.Anything, mock.Anything).Return(nil, c.grpcErr)

			err := gw.Unlock(context.Background(), "lnd-0", []byte(unlockPassword))

			assert.Error(t, err)
			for _, s := range sentinels {
				if s == c.sentinel {
					assert.ErrorIs(t, err, s)
				} else {
					assert.NotErrorIs(t, err, s)
				}
			}
			assert.NotContains(t, err.Error(), unlockPassword)
		})
	}
}

func TestUnlock_FactoryError(t *testing.T) {
	gw, factory := newGateway()
	factory.Err = errors.New("cannot dial")

	err := gw.Unlock(context.Background(), "lnd-0", []byte(unlockPassword))

	assert.Error(t, err)
	for _, s := range sentinels {
		assert.NotErrorIs(t, err, s)
	}
	assert.NotContains(t, err.Error(), unlockPassword)
	factory.WalletUnlocker.AssertNotCalled(t, "UnlockWallet", mock.Anything, mock.Anything)
}
