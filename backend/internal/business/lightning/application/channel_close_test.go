package application_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Lmare/lightning-playground/backend/internal/business/lightning/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var closeTxid = strings.Repeat("cd", 32)

func chanWith(st domain.ChannelStatus) domain.Channel {
	return domain.Channel{Point: domain.ChannelPoint{FundingTxID: closeTxid, OutputIndex: 1}, Status: st, CapacitySat: 100}
}

func TestListChannels_Delegates(t *testing.T) {
	svc, channels, _, _ := newChannelService()
	want := []domain.Channel{chanWith(domain.ChannelActive)}
	channels.On("ListChannels", mock.Anything, domain.NodeID("lnd-0")).Return(want, nil).Once()

	got, err := svc.ListChannels(context.Background(), "lnd-0")

	assert.NoError(t, err)
	assert.Equal(t, want, got)

	boom := errors.New("boom")
	channels.On("ListChannels", mock.Anything, mock.Anything).Return(nil, boom).Once()
	_, err = svc.ListChannels(context.Background(), "lnd-0")
	assert.ErrorIs(t, err, boom)
}

func TestCloseChannel_InvalidPoint(t *testing.T) {
	svc, channels, _, _ := newChannelService()

	_, err := svc.CloseChannel(context.Background(), "lnd-0", "garbage", false)

	assert.ErrorIs(t, err, domain.ErrInvalidChannelPoint)
	channels.AssertNotCalled(t, "ListChannels", mock.Anything, mock.Anything)
	channels.AssertNotCalled(t, "CloseChannel", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestCloseChannel_UnknownPoint(t *testing.T) {
	svc, channels, _, _ := newChannelService()
	channels.On("ListChannels", mock.Anything, mock.Anything).Return([]domain.Channel{chanWith(domain.ChannelActive)}, nil)

	_, err := svc.CloseChannel(context.Background(), "lnd-0", closeTxid+":2", false)

	assert.ErrorIs(t, err, domain.ErrChannelNotFound)
	channels.AssertNotCalled(t, "CloseChannel", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestCloseChannel_NotClosable(t *testing.T) {
	for _, st := range []domain.ChannelStatus{domain.ChannelPendingOpen, domain.ChannelClosing, domain.ChannelForceClosing} {
		svc, channels, _, _ := newChannelService()
		channels.On("ListChannels", mock.Anything, mock.Anything).Return([]domain.Channel{chanWith(st)}, nil)

		_, err := svc.CloseChannel(context.Background(), "lnd-0", closeTxid+":1", true)

		assert.ErrorIs(t, err, domain.ErrChannelNotClosable, st)
		channels.AssertNotCalled(t, "CloseChannel", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	}
}

func TestCloseChannel_InactiveCooperativeVsForce(t *testing.T) {
	svc, channels, _, _ := newChannelService()
	channels.On("ListChannels", mock.Anything, mock.Anything).Return([]domain.Channel{chanWith(domain.ChannelInactive)}, nil)

	_, err := svc.CloseChannel(context.Background(), "lnd-0", closeTxid+":1", false)
	assert.ErrorIs(t, err, domain.ErrChannelPeerOffline)
	channels.AssertNotCalled(t, "CloseChannel", mock.Anything, mock.Anything, mock.Anything, mock.Anything)

	point := domain.ChannelPoint{FundingTxID: closeTxid, OutputIndex: 1}
	channels.On("CloseChannel", mock.Anything, domain.NodeID("lnd-0"), point, true).Return("closetx", nil).Once()
	txid, err := svc.CloseChannel(context.Background(), "lnd-0", closeTxid+":1", true)
	assert.NoError(t, err)
	assert.Equal(t, "closetx", txid)
}

func TestCloseChannel_ActiveCooperative(t *testing.T) {
	svc, channels, _, _ := newChannelService()
	channels.On("ListChannels", mock.Anything, mock.Anything).Return([]domain.Channel{chanWith(domain.ChannelActive)}, nil)
	point := domain.ChannelPoint{FundingTxID: closeTxid, OutputIndex: 1}
	channels.On("CloseChannel", mock.Anything, domain.NodeID("lnd-0"), point, false).Return("closetx", nil).Once()

	txid, err := svc.CloseChannel(context.Background(), "lnd-0", closeTxid+":1", false)

	assert.NoError(t, err)
	assert.Equal(t, "closetx", txid)
	channels.AssertExpectations(t)
}

func TestCloseChannel_ListErrorAndCloseError(t *testing.T) {
	svc, channels, _, _ := newChannelService()
	boom := errors.New("list failed")
	channels.On("ListChannels", mock.Anything, mock.Anything).Return(nil, boom).Once()

	_, err := svc.CloseChannel(context.Background(), "lnd-0", closeTxid+":1", false)
	assert.ErrorIs(t, err, boom)
	channels.AssertNotCalled(t, "CloseChannel", mock.Anything, mock.Anything, mock.Anything, mock.Anything)

	channels.On("ListChannels", mock.Anything, mock.Anything).Return([]domain.Channel{chanWith(domain.ChannelActive)}, nil)
	closeErr := errors.New("close failed")
	channels.On("CloseChannel", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("", closeErr)
	_, err = svc.CloseChannel(context.Background(), "lnd-0", closeTxid+":1", false)
	assert.ErrorIs(t, err, closeErr)
}
