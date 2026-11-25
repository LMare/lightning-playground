package domain_test

import (
	"testing"

	"github.com/Lmare/lightning-playground/api/internal/business/lightning/domain"
	"github.com/stretchr/testify/assert"
)

func TestOnChainBalance_TotalSat(t *testing.T) {
	b := domain.OnChainBalance{ConfirmedSat: 1000, UnconfirmedSat: 250, LockedSat: 77, ReservedAnchorSat: 33}
	assert.Equal(t, int64(1250), b.TotalSat())
}

func TestChannelBalance_TotalLocalSat(t *testing.T) {
	b := domain.ChannelBalance{
		LocalSat: 600, RemoteSat: 390,
		PendingOpenLocalSat: 50, PendingOpenRemoteSat: 7,
		UnsettledLocalSat: 10, UnsettledRemoteSat: 3,
	}
	assert.Equal(t, int64(660), b.TotalLocalSat())
}

func TestBalance_TotalSat(t *testing.T) {
	b := domain.Balance{
		OnChain:  domain.OnChainBalance{ConfirmedSat: 1000, UnconfirmedSat: 250},
		Channels: domain.ChannelBalance{LocalSat: 600, RemoteSat: 390, PendingOpenLocalSat: 50, UnsettledLocalSat: 10},
	}
	assert.Equal(t, int64(1910), b.TotalSat())
	assert.Equal(t, int64(0), domain.Balance{}.TotalSat())
}
