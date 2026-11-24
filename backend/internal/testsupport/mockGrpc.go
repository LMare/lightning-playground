package testsupport

import (
	"context"
	"io"

	lnrpc "github.com/Lmare/lightning-playground/backend/gRPC/github.com/lightningnetwork/lnd/lnrpc"
	routerrpc "github.com/Lmare/lightning-playground/backend/gRPC/github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	lnd "github.com/Lmare/lightning-playground/backend/internal/infra/grpc/lnd"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

// MockLightningClient mocks the LND Lightning service. Only the methods used by the
// application are implemented; calling another one panics (nil embedded interface).
type MockLightningClient struct {
	lnrpc.LightningClient
	mock.Mock
}

func (m *MockLightningClient) GetInfo(ctx context.Context, in *lnrpc.GetInfoRequest, _ ...grpc.CallOption) (*lnrpc.GetInfoResponse, error) {
	args := m.Called(ctx, in)
	resp, _ := args.Get(0).(*lnrpc.GetInfoResponse)
	return resp, args.Error(1)
}

func (m *MockLightningClient) ConnectPeer(ctx context.Context, in *lnrpc.ConnectPeerRequest, _ ...grpc.CallOption) (*lnrpc.ConnectPeerResponse, error) {
	args := m.Called(ctx, in)
	resp, _ := args.Get(0).(*lnrpc.ConnectPeerResponse)
	return resp, args.Error(1)
}

func (m *MockLightningClient) OpenChannelSync(ctx context.Context, in *lnrpc.OpenChannelRequest, _ ...grpc.CallOption) (*lnrpc.ChannelPoint, error) {
	args := m.Called(ctx, in)
	resp, _ := args.Get(0).(*lnrpc.ChannelPoint)
	return resp, args.Error(1)
}

func (m *MockLightningClient) AddInvoice(ctx context.Context, in *lnrpc.Invoice, _ ...grpc.CallOption) (*lnrpc.AddInvoiceResponse, error) {
	args := m.Called(ctx, in)
	resp, _ := args.Get(0).(*lnrpc.AddInvoiceResponse)
	return resp, args.Error(1)
}

func (m *MockLightningClient) SetAlias(ctx context.Context, in *lnrpc.SetAliasRequest, _ ...grpc.CallOption) (*lnrpc.SetAliasResponse, error) {
	args := m.Called(ctx, in)
	resp, _ := args.Get(0).(*lnrpc.SetAliasResponse)
	return resp, args.Error(1)
}

func (m *MockLightningClient) SetColor(ctx context.Context, in *lnrpc.SetColorRequest, _ ...grpc.CallOption) (*lnrpc.SetColorResponse, error) {
	args := m.Called(ctx, in)
	resp, _ := args.Get(0).(*lnrpc.SetColorResponse)
	return resp, args.Error(1)
}

// MockStateClient mocks the LND State service.
type MockStateClient struct {
	lnrpc.StateClient
	mock.Mock
}

func (m *MockStateClient) GetState(ctx context.Context, in *lnrpc.GetStateRequest, _ ...grpc.CallOption) (*lnrpc.GetStateResponse, error) {
	args := m.Called(ctx, in)
	resp, _ := args.Get(0).(*lnrpc.GetStateResponse)
	return resp, args.Error(1)
}

// MockWalletUnlockerClient mocks the LND WalletUnlocker service.
type MockWalletUnlockerClient struct {
	lnrpc.WalletUnlockerClient
	mock.Mock
}

func (m *MockWalletUnlockerClient) UnlockWallet(ctx context.Context, in *lnrpc.UnlockWalletRequest, _ ...grpc.CallOption) (*lnrpc.UnlockWalletResponse, error) {
	args := m.Called(ctx, in)
	resp, _ := args.Get(0).(*lnrpc.UnlockWalletResponse)
	return resp, args.Error(1)
}

// MockRouterClient mocks the LND Router service.
type MockRouterClient struct {
	routerrpc.RouterClient
	mock.Mock
}

func (m *MockRouterClient) EstimateRouteFee(ctx context.Context, in *routerrpc.RouteFeeRequest, _ ...grpc.CallOption) (*routerrpc.RouteFeeResponse, error) {
	args := m.Called(ctx, in)
	resp, _ := args.Get(0).(*routerrpc.RouteFeeResponse)
	return resp, args.Error(1)
}

func (m *MockRouterClient) SendPaymentV2(ctx context.Context, in *routerrpc.SendPaymentRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[lnrpc.Payment], error) {
	args := m.Called(ctx, in)
	resp, _ := args.Get(0).(grpc.ServerStreamingClient[lnrpc.Payment])
	return resp, args.Error(1)
}

// PaymentStream is a canned grpc payment stream: it yields the payments then io.EOF.
type PaymentStream struct {
	grpc.ServerStreamingClient[lnrpc.Payment]
	Payments []*lnrpc.Payment
	next     int
}

func (s *PaymentStream) Recv() (*lnrpc.Payment, error) {
	if s.next >= len(s.Payments) {
		return nil, io.EOF
	}
	p := s.Payments[s.next]
	s.next++
	return p, nil
}

func (m *MockLightningClient) ListChannels(ctx context.Context, in *lnrpc.ListChannelsRequest, _ ...grpc.CallOption) (*lnrpc.ListChannelsResponse, error) {
	args := m.Called(ctx, in)
	resp, _ := args.Get(0).(*lnrpc.ListChannelsResponse)
	return resp, args.Error(1)
}

func (m *MockLightningClient) PendingChannels(ctx context.Context, in *lnrpc.PendingChannelsRequest, _ ...grpc.CallOption) (*lnrpc.PendingChannelsResponse, error) {
	args := m.Called(ctx, in)
	resp, _ := args.Get(0).(*lnrpc.PendingChannelsResponse)
	return resp, args.Error(1)
}

func (m *MockLightningClient) CloseChannel(ctx context.Context, in *lnrpc.CloseChannelRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[lnrpc.CloseStatusUpdate], error) {
	args := m.Called(ctx, in)
	resp, _ := args.Get(0).(grpc.ServerStreamingClient[lnrpc.CloseStatusUpdate])
	return resp, args.Error(1)
}

// CloseStream is a canned grpc close stream: it yields the updates then io.EOF.
type CloseStream struct {
	grpc.ServerStreamingClient[lnrpc.CloseStatusUpdate]
	Updates []*lnrpc.CloseStatusUpdate
	RecvErr error // when set, Recv returns it immediately
	next    int
}

func (s *CloseStream) Recv() (*lnrpc.CloseStatusUpdate, error) {
	if s.RecvErr != nil {
		return nil, s.RecvErr
	}
	if s.next >= len(s.Updates) {
		return nil, io.EOF
	}
	u := s.Updates[s.next]
	s.next++
	return u, nil
}

// GrpcClientFactoryMock hands out the given mocked LND clients for any node.
type GrpcClientFactoryMock struct {
	Lightning *MockLightningClient
	Router    *MockRouterClient
	// State is the default State service mock.
	State *MockStateClient
	// StateByAddress overrides State for a given node address (e.g. "lnd-1.lnd.test:10009").
	StateByAddress map[string]*MockStateClient
	// WalletUnlocker is the WalletUnlocker service mock.
	WalletUnlocker *MockWalletUnlockerClient
	// Connections records the connection info requested by the gateway.
	Connections []lnd.NodeConnection
	// Err makes NewClient fail when set.
	Err error
}

func NewGrpcClientFactoryMock() *GrpcClientFactoryMock {
	return &GrpcClientFactoryMock{Lightning: &MockLightningClient{}, Router: &MockRouterClient{}, State: &MockStateClient{}, WalletUnlocker: &MockWalletUnlockerClient{}}
}

func (f *GrpcClientFactoryMock) NewClient(conn lnd.NodeConnection) (lnd.GrpcClient, error) {
	f.Connections = append(f.Connections, conn)
	if f.Err != nil {
		return nil, f.Err
	}
	var state lnrpc.StateClient = f.State
	if st, ok := f.StateByAddress[conn.Address]; ok {
		state = st
	}
	return &mockClient{lightning: f.Lightning, router: f.Router, state: state, unlocker: f.WalletUnlocker}, nil
}

type mockClient struct {
	lightning lnrpc.LightningClient
	router    routerrpc.RouterClient
	state     lnrpc.StateClient
	unlocker  lnrpc.WalletUnlockerClient
}

func (c *mockClient) GetLightningClient() lnrpc.LightningClient           { return c.lightning }
func (c *mockClient) GetRouterClient() routerrpc.RouterClient             { return c.router }
func (c *mockClient) GetStateClient() lnrpc.StateClient                   { return c.state }
func (c *mockClient) GetWalletUnlockerClient() lnrpc.WalletUnlockerClient { return c.unlocker }
func (c *mockClient) Close() error                                        { return nil }

var _ lnd.GrpcClientFactory = (*GrpcClientFactoryMock)(nil)
