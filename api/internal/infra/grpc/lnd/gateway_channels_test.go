package lnd_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	lnrpc "github.com/Lmare/lightning-playground/api/gRPC/github.com/lightningnetwork/lnd/lnrpc"
	"github.com/Lmare/lightning-playground/api/internal/business/lightning/domain"
	"github.com/Lmare/lightning-playground/api/internal/testsupport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	tx1 = strings.Repeat("11", 32)
	tx2 = strings.Repeat("22", 32)
	tx3 = strings.Repeat("33", 32)
	tx4 = strings.Repeat("44", 32)
	tx5 = strings.Repeat("55", 32)
)

func pending(point, pub string, capacity, local, remote int64) *lnrpc.PendingChannelsResponse_PendingChannel {
	return &lnrpc.PendingChannelsResponse_PendingChannel{RemoteNodePub: pub, ChannelPoint: point, Capacity: capacity, LocalBalance: local, RemoteBalance: remote}
}

func TestListChannels_Mapping(t *testing.T) {
	gw, factory := newGateway()
	factory.Lightning.On("ListChannels", mock.Anything, mock.Anything).Return(&lnrpc.ListChannelsResponse{Channels: []*lnrpc.Channel{
		{ChannelPoint: tx1 + ":0", RemotePubkey: "pubA", Capacity: 1000, LocalBalance: 600, RemoteBalance: 390, Active: true},
		{ChannelPoint: tx2 + ":1", RemotePubkey: "pubB", Capacity: 2000, LocalBalance: 1, RemoteBalance: 2, Active: false},
	}}, nil)
	factory.Lightning.On("PendingChannels", mock.Anything, mock.Anything).Return(&lnrpc.PendingChannelsResponse{
		PendingOpenChannels:         []*lnrpc.PendingChannelsResponse_PendingOpenChannel{{Channel: pending(tx3+":0", "pubC", 300, 200, 100)}},
		WaitingCloseChannels:        []*lnrpc.PendingChannelsResponse_WaitingCloseChannel{{Channel: pending(tx4+":2", "pubD", 400, 300, 100)}},
		PendingForceClosingChannels: []*lnrpc.PendingChannelsResponse_ForceClosedChannel{{Channel: pending(tx5+":3", "pubE", 500, 400, 100)}},
	}, nil)

	got, err := gw.ListChannels(context.Background(), "lnd-0")

	assert.NoError(t, err)
	assert.Equal(t, []domain.Channel{
		{Point: domain.ChannelPoint{FundingTxID: tx1, OutputIndex: 0}, RemotePubKey: "pubA", CapacitySat: 1000, LocalBalanceSat: 600, RemoteBalanceSat: 390, Status: domain.ChannelActive},
		{Point: domain.ChannelPoint{FundingTxID: tx2, OutputIndex: 1}, RemotePubKey: "pubB", CapacitySat: 2000, LocalBalanceSat: 1, RemoteBalanceSat: 2, Status: domain.ChannelInactive},
		{Point: domain.ChannelPoint{FundingTxID: tx3, OutputIndex: 0}, RemotePubKey: "pubC", CapacitySat: 300, LocalBalanceSat: 200, RemoteBalanceSat: 100, Status: domain.ChannelPendingOpen},
		{Point: domain.ChannelPoint{FundingTxID: tx4, OutputIndex: 2}, RemotePubKey: "pubD", CapacitySat: 400, LocalBalanceSat: 300, RemoteBalanceSat: 100, Status: domain.ChannelClosing},
		{Point: domain.ChannelPoint{FundingTxID: tx5, OutputIndex: 3}, RemotePubKey: "pubE", CapacitySat: 500, LocalBalanceSat: 400, RemoteBalanceSat: 100, Status: domain.ChannelForceClosing},
	}, got)
}

func TestListChannels_Empty(t *testing.T) {
	gw, factory := newGateway()
	factory.Lightning.On("ListChannels", mock.Anything, mock.Anything).Return(&lnrpc.ListChannelsResponse{}, nil)
	factory.Lightning.On("PendingChannels", mock.Anything, mock.Anything).Return(&lnrpc.PendingChannelsResponse{}, nil)

	got, err := gw.ListChannels(context.Background(), "lnd-0")

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Empty(t, got)
}

func TestListChannels_Failures(t *testing.T) {
	boom := errors.New("boom")
	okList := &lnrpc.ListChannelsResponse{}
	okPending := &lnrpc.PendingChannelsResponse{}

	t.Run("list rpc fails", func(t *testing.T) {
		gw, f := newGateway()
		f.Lightning.On("ListChannels", mock.Anything, mock.Anything).Return(nil, boom)
		f.Lightning.On("PendingChannels", mock.Anything, mock.Anything).Return(okPending, nil).Maybe()
		_, err := gw.ListChannels(context.Background(), "lnd-0")
		assert.ErrorIs(t, err, boom)
	})
	t.Run("pending rpc fails", func(t *testing.T) {
		gw, f := newGateway()
		f.Lightning.On("ListChannels", mock.Anything, mock.Anything).Return(okList, nil)
		f.Lightning.On("PendingChannels", mock.Anything, mock.Anything).Return(nil, boom)
		_, err := gw.ListChannels(context.Background(), "lnd-0")
		assert.ErrorIs(t, err, boom)
	})
	t.Run("malformed open channel point", func(t *testing.T) {
		gw, f := newGateway()
		f.Lightning.On("ListChannels", mock.Anything, mock.Anything).Return(&lnrpc.ListChannelsResponse{Channels: []*lnrpc.Channel{{ChannelPoint: "bad"}}}, nil)
		f.Lightning.On("PendingChannels", mock.Anything, mock.Anything).Return(okPending, nil).Maybe()
		_, err := gw.ListChannels(context.Background(), "lnd-0")
		assert.Error(t, err)
	})
	t.Run("malformed pending channel point", func(t *testing.T) {
		gw, f := newGateway()
		f.Lightning.On("ListChannels", mock.Anything, mock.Anything).Return(okList, nil)
		f.Lightning.On("PendingChannels", mock.Anything, mock.Anything).Return(&lnrpc.PendingChannelsResponse{
			PendingOpenChannels: []*lnrpc.PendingChannelsResponse_PendingOpenChannel{{Channel: pending("bad", "p", 1, 1, 0)}},
		}, nil)
		_, err := gw.ListChannels(context.Background(), "lnd-0")
		assert.Error(t, err)
	})
	t.Run("dial error", func(t *testing.T) {
		gw, f := newGateway()
		f.Err = boom
		_, err := gw.ListChannels(context.Background(), "lnd-0")
		assert.ErrorIs(t, err, boom)
	})
}

// internal byte order is the reverse of the displayed txid.
func internalBytes(displayHex string) []byte {
	out := make([]byte, 0, 32)
	for i := 31; i >= 0; i-- {
		var b byte
		for _, c := range displayHex[i*2 : i*2+2] {
			b <<= 4
			if c >= 'a' {
				b |= byte(c-'a') + 10
			} else {
				b |= byte(c - '0')
			}
		}
		out = append(out, b)
	}
	return out
}

func displayTxid() string {
	// 00 01 02 ... 1f displayed; internal order is reversed
	var sb strings.Builder
	for i := 0; i < 32; i++ {
		sb.WriteString(string("0123456789abcdef"[i>>4]))
		sb.WriteString(string("0123456789abcdef"[i&15]))
	}
	return sb.String()
}

func TestCloseChannel_ClosePending(t *testing.T) {
	gw, factory := newGateway()
	want := displayTxid()
	stream := &testsupport.CloseStream{Updates: []*lnrpc.CloseStatusUpdate{
		{Update: &lnrpc.CloseStatusUpdate_ClosePending{ClosePending: &lnrpc.PendingUpdate{Txid: internalBytes(want)}}},
	}}
	factory.Lightning.On("CloseChannel", mock.Anything, mock.MatchedBy(func(r *lnrpc.CloseChannelRequest) bool {
		return r.GetChannelPoint().GetFundingTxidStr() == tx1 && r.GetChannelPoint().GetOutputIndex() == 5 && r.Force
	})).Return(stream, nil)

	txid, err := gw.CloseChannel(context.Background(), "lnd-0", domain.ChannelPoint{FundingTxID: tx1, OutputIndex: 5}, true)

	assert.NoError(t, err)
	assert.Equal(t, want, txid)
	factory.Lightning.AssertNumberOfCalls(t, "CloseChannel", 1)
}

func TestCloseChannel_CooperativeSendsForceFalse(t *testing.T) {
	gw, factory := newGateway()
	stream := &testsupport.CloseStream{Updates: []*lnrpc.CloseStatusUpdate{
		{Update: &lnrpc.CloseStatusUpdate_ClosePending{ClosePending: &lnrpc.PendingUpdate{Txid: internalBytes(tx2)}}},
	}}
	factory.Lightning.On("CloseChannel", mock.Anything, mock.MatchedBy(func(r *lnrpc.CloseChannelRequest) bool {
		return !r.Force && r.GetChannelPoint().GetOutputIndex() == 0
	})).Return(stream, nil)

	_, err := gw.CloseChannel(context.Background(), "lnd-0", domain.ChannelPoint{FundingTxID: tx1, OutputIndex: 0}, false)

	assert.NoError(t, err)
}

func TestCloseChannel_ChanClose(t *testing.T) {
	gw, factory := newGateway()
	stream := &testsupport.CloseStream{Updates: []*lnrpc.CloseStatusUpdate{
		{Update: &lnrpc.CloseStatusUpdate_ChanClose{ChanClose: &lnrpc.ChannelCloseUpdate{ClosingTxid: internalBytes(tx3), Success: true}}},
	}}
	factory.Lightning.On("CloseChannel", mock.Anything, mock.Anything).Return(stream, nil)

	txid, err := gw.CloseChannel(context.Background(), "lnd-0", domain.ChannelPoint{FundingTxID: tx1}, false)

	assert.NoError(t, err)
	assert.Equal(t, tx3, txid)
}

func TestCloseChannel_Failures(t *testing.T) {
	boom := errors.New("boom")
	point := domain.ChannelPoint{FundingTxID: tx1}

	t.Run("rpc error", func(t *testing.T) {
		gw, f := newGateway()
		f.Lightning.On("CloseChannel", mock.Anything, mock.Anything).Return(nil, boom)
		_, err := gw.CloseChannel(context.Background(), "lnd-0", point, false)
		assert.ErrorIs(t, err, boom)
	})
	t.Run("recv error", func(t *testing.T) {
		gw, f := newGateway()
		f.Lightning.On("CloseChannel", mock.Anything, mock.Anything).Return(&testsupport.CloseStream{RecvErr: boom}, nil)
		_, err := gw.CloseChannel(context.Background(), "lnd-0", point, false)
		assert.ErrorIs(t, err, boom)
	})
	t.Run("empty stream", func(t *testing.T) {
		gw, f := newGateway()
		f.Lightning.On("CloseChannel", mock.Anything, mock.Anything).Return(&testsupport.CloseStream{}, nil)
		_, err := gw.CloseChannel(context.Background(), "lnd-0", point, false)
		assert.Error(t, err)
	})
	t.Run("unexpected update type", func(t *testing.T) {
		gw, f := newGateway()
		f.Lightning.On("CloseChannel", mock.Anything, mock.Anything).Return(&testsupport.CloseStream{Updates: []*lnrpc.CloseStatusUpdate{
			{Update: &lnrpc.CloseStatusUpdate_CloseInstant{CloseInstant: &lnrpc.InstantUpdate{}}},
		}}, nil)
		_, err := gw.CloseChannel(context.Background(), "lnd-0", point, false)
		assert.Error(t, err)
	})
	t.Run("dial error", func(t *testing.T) {
		gw, f := newGateway()
		f.Err = boom
		_, err := gw.CloseChannel(context.Background(), "lnd-0", point, false)
		assert.ErrorIs(t, err, boom)
	})
}
