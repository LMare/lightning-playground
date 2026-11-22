package lnd_test

import (
	"context"
	"encoding/hex"
	"errors"
	"io"
	"testing"

	config "github.com/Lmare/lightning-playground"
	lnrpc "github.com/Lmare/lightning-playground/backend/gRPC/github.com/lightningnetwork/lnd/lnrpc"
	routerrpc "github.com/Lmare/lightning-playground/backend/gRPC/github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"github.com/Lmare/lightning-playground/backend/internal/business/lightning/domain"
	lnd "github.com/Lmare/lightning-playground/backend/internal/infra/grpc/lnd"
	"github.com/Lmare/lightning-playground/backend/internal/testsupport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newGateway() (*lnd.Gateway, *testsupport.GrpcClientFactoryMock) {
	factory := testsupport.NewGrpcClientFactoryMock()
	resolver := lnd.NewConfigConnectionResolver(&config.Config{NodeStorage: "/nodes/", LndNetwork: "simnet", LndDomain: "lnd.test"})
	return lnd.NewGateway(factory, resolver), factory
}

func TestConnectionResolver(t *testing.T) {
	resolver := lnd.NewConfigConnectionResolver(&config.Config{NodeStorage: "/nodes/", LndNetwork: "simnet", LndDomain: "lnd.test"})

	assert.Equal(t, lnd.NodeConnection{
		TLSCertPath:  "/nodes/lnd-0/tls.cert",
		MacaroonPath: "/nodes/lnd-0/data/chain/bitcoin/simnet/admin.macaroon",
		Address:      "lnd-0.lnd.test:10009",
	}, resolver.Resolve("lnd-0"))
}

func TestGetInfo_MapsTheLndResponse(t *testing.T) {
	gw, factory := newGateway()
	factory.Lightning.On("GetInfo", mock.Anything, mock.Anything).Return(&lnrpc.GetInfoResponse{
		Alias: "alice", Color: "#112233", NumPeers: 2, NumActiveChannels: 1, BlockHeight: 10,
		Uris: []string{"k@h:9735"}, SyncedToChain: true,
		Chains: []*lnrpc.Chain{{Chain: "bitcoin", Network: "simnet"}},
	}, nil)

	info, err := gw.GetInfo(context.Background(), "lnd-0")

	assert.NoError(t, err)
	assert.Equal(t, domain.NodeInfo{
		Alias: "alice", Color: "#112233", NumPeers: 2, NumActiveChannels: 1, BlockHeight: 10,
		Network: "simnet", URIs: []string{"k@h:9735"}, SyncedToChain: true,
	}, info)
	assert.Equal(t, "lnd-0.lnd.test:10009", factory.Connections[0].Address)
}

func TestGetInfo_NoChainDoesNotPanic(t *testing.T) {
	gw, factory := newGateway()
	factory.Lightning.On("GetInfo", mock.Anything, mock.Anything).Return(&lnrpc.GetInfoResponse{}, nil)

	info, err := gw.GetInfo(context.Background(), "lnd-0")

	assert.NoError(t, err)
	assert.Equal(t, "", info.Network)
}

func TestGetSummary_Errors(t *testing.T) {
	gw, factory := newGateway()
	factory.Lightning.On("GetInfo", mock.Anything, mock.Anything).Return(nil, errors.New("rpc down"))
	_, err := gw.GetSummary(context.Background(), "lnd-0")
	assert.Error(t, err)

	factory.Err = errors.New("cannot dial")
	_, err = gw.GetSummary(context.Background(), "lnd-0")
	assert.Error(t, err)
}

func TestAddPeer(t *testing.T) {
	gw, factory := newGateway()
	factory.Lightning.On("ConnectPeer", mock.Anything, &lnrpc.ConnectPeerRequest{
		Addr: &lnrpc.LightningAddress{Pubkey: "02ab", Host: "h:9735"}, Perm: true,
	}).Return(&lnrpc.ConnectPeerResponse{}, nil)

	assert.NoError(t, gw.AddPeer(context.Background(), "lnd-0", domain.PeerAddress{PubKey: "02ab", Host: "h:9735"}))
	factory.Lightning.AssertExpectations(t)
}

func TestOpenChannel(t *testing.T) {
	gw, factory := newGateway()
	factory.Lightning.On("OpenChannelSync", mock.Anything, mock.MatchedBy(func(r *lnrpc.OpenChannelRequest) bool {
		return r.LocalFundingAmount == 5000 && hex.EncodeToString(r.NodePubkey) == "02ab"
	})).Return(&lnrpc.ChannelPoint{}, nil)

	assert.NoError(t, gw.OpenChannel(context.Background(), "lnd-0", "02ab", 5000))
	assert.Error(t, gw.OpenChannel(context.Background(), "lnd-0", "not-hex", 5000))
}

func TestUpdateAppearance(t *testing.T) {
	gw, factory := newGateway()
	factory.Lightning.On("SetAlias", mock.Anything, &lnrpc.SetAliasRequest{Alias: "bob"}).Return(&lnrpc.SetAliasResponse{}, nil)
	factory.Lightning.On("SetColor", mock.Anything, &lnrpc.SetColorRequest{Color: "#000000"}).Return(&lnrpc.SetColorResponse{}, nil)

	assert.NoError(t, gw.UpdateAppearance(context.Background(), "lnd-0", "bob", "#000000"))
	factory.Lightning.AssertExpectations(t)
}

func TestCreateInvoice(t *testing.T) {
	gw, factory := newGateway()
	factory.Lightning.On("AddInvoice", mock.Anything, &lnrpc.Invoice{Expiry: 300, Memo: "coffee", Value: 100}).
		Return(&lnrpc.AddInvoiceResponse{PaymentRequest: "lnbc1"}, nil)

	pr, err := gw.CreateInvoice(context.Background(), "lnd-0", "coffee", 100)

	assert.NoError(t, err)
	assert.Equal(t, "lnbc1", pr)
}

func TestPayInvoice_StreamsDomainUpdatesAndAppliesAFeeMargin(t *testing.T) {
	gw, factory := newGateway()
	factory.Router.On("EstimateRouteFee", mock.Anything, mock.Anything).Return(&routerrpc.RouteFeeResponse{RoutingFeeMsat: 1000}, nil)
	factory.Router.On("SendPaymentV2", mock.Anything, mock.MatchedBy(func(r *routerrpc.SendPaymentRequest) bool {
		return r.PaymentRequest == "lnbc1" && r.FeeLimitMsat == 1200 // +20 %
	})).Return(&testsupport.PaymentStream{Payments: []*lnrpc.Payment{
		{ValueSat: 100, Status: lnrpc.Payment_IN_FLIGHT},
		{ValueSat: 100, Status: lnrpc.Payment_SUCCEEDED},
	}}, nil)

	progress, err := gw.PayInvoice(context.Background(), "lnd-0", "lnbc1")
	assert.NoError(t, err)

	first, err := progress.Recv()
	assert.NoError(t, err)
	assert.Equal(t, domain.PaymentUpdate{AmountSat: 100, Status: domain.PaymentInFlight}, first)
	second, _ := progress.Recv()
	assert.Equal(t, domain.PaymentSucceeded, second.Status)
	_, err = progress.Recv()
	assert.ErrorIs(t, err, io.EOF)
	assert.NoError(t, progress.Close())
}

func TestPayInvoice_FeeEstimationFallback(t *testing.T) {
	gw, factory := newGateway()
	factory.Router.On("EstimateRouteFee", mock.Anything, mock.Anything).Return(nil, errors.New("no route"))
	factory.Router.On("SendPaymentV2", mock.Anything, mock.MatchedBy(func(r *routerrpc.SendPaymentRequest) bool {
		return r.FeeLimitMsat == 100000
	})).Return(&testsupport.PaymentStream{}, nil)

	_, err := gw.PayInvoice(context.Background(), "lnd-0", "lnbc1")

	assert.NoError(t, err)
}

func TestGetWalletState_MapsEveryLndState(t *testing.T) {
	tests := []struct {
		in   lnrpc.WalletState
		want domain.WalletState
	}{
		{lnrpc.WalletState_NON_EXISTING, domain.WalletNonExisting},
		{lnrpc.WalletState_LOCKED, domain.WalletLocked},
		{lnrpc.WalletState_UNLOCKED, domain.WalletUnlocked},
		{lnrpc.WalletState_RPC_ACTIVE, domain.WalletRPCActive},
		{lnrpc.WalletState_SERVER_ACTIVE, domain.WalletServerActive},
		{lnrpc.WalletState_WAITING_TO_START, domain.WalletWaitingToStart},
		{lnrpc.WalletState(42), domain.WalletUnknown},
	}
	for _, tt := range tests {
		t.Run(string(tt.want), func(t *testing.T) {
			gw, factory := newGateway()
			factory.State.On("GetState", mock.Anything, mock.Anything).Return(&lnrpc.GetStateResponse{State: tt.in}, nil)

			state, err := gw.GetWalletState(context.Background(), "lnd-0")

			assert.NoError(t, err)
			assert.Equal(t, tt.want, state)
			assert.Equal(t, "lnd-0.lnd.test:10009", factory.Connections[0].Address)
		})
	}
}

func TestGetWalletState_GetStateError(t *testing.T) {
	gw, factory := newGateway()
	factory.State.On("GetState", mock.Anything, mock.Anything).Return(nil, errors.New("unavailable"))

	_, err := gw.GetWalletState(context.Background(), "lnd-0")

	assert.Error(t, err)
}

func TestGetWalletState_FactoryError(t *testing.T) {
	gw, factory := newGateway()
	factory.Err = errors.New("no macaroon")

	_, err := gw.GetWalletState(context.Background(), "lnd-0")

	assert.Error(t, err)
	factory.State.AssertNotCalled(t, "GetState", mock.Anything, mock.Anything)
}
