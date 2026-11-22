package lnd

import (
	"context"
	"encoding/hex"
	"fmt"

	lnrpc "github.com/Lmare/lightning-playground/backend/gRPC/github.com/lightningnetwork/lnd/lnrpc"
	routerrpc "github.com/Lmare/lightning-playground/backend/gRPC/github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"github.com/Lmare/lightning-playground/backend/internal/business/lightning/domain"
	"github.com/Lmare/lightning-playground/backend/internal/business/lightning/port"
	exception "github.com/Lmare/lightning-playground/backend/internal/shared/exception"
	"google.golang.org/grpc"
)

// Gateway is the outbound adapter talking to LND nodes over gRPC.
// It implements the NodeInspector, ChannelGateway and PaymentGateway ports.
type Gateway struct {
	factory  GrpcClientFactory
	resolver ConnectionResolver
}

var (
	_ port.NodeInspector  = (*Gateway)(nil)
	_ port.ChannelGateway = (*Gateway)(nil)
	_ port.PaymentGateway = (*Gateway)(nil)
)

func NewGateway(factory GrpcClientFactory, resolver ConnectionResolver) *Gateway {
	return &Gateway{factory: factory, resolver: resolver}
}

func (g *Gateway) open(id domain.NodeID) (GrpcClient, error) {
	client, err := g.factory.NewClient(g.resolver.Resolve(id))
	if err != nil {
		return nil, exception.NewError(fmt.Sprintf("Unable to open dial with Node[%s]", id), err)
	}
	return client, nil
}

// ---------------- NodeInspector ----------------

func (g *Gateway) GetSummary(ctx context.Context, id domain.NodeID) (domain.NodeSummary, error) {
	resp, err := g.getInfo(ctx, id)
	if err != nil {
		return domain.NodeSummary{}, err
	}
	return domain.NodeSummary{ID: id, Alias: resp.GetAlias(), Color: resp.GetColor()}, nil
}

func (g *Gateway) GetWalletState(ctx context.Context, id domain.NodeID) (domain.WalletState, error) {
	client, err := g.open(id)
	if err != nil {
		return "", err
	}
	defer client.Close()

	resp, err := client.GetStateClient().GetState(ctx, &lnrpc.GetStateRequest{})
	if err != nil {
		return "", exception.NewError(fmt.Sprintf("Unable to get wallet state of Node[%s]", id), err)
	}
	return toWalletState(resp.GetState()), nil
}

func toWalletState(s lnrpc.WalletState) domain.WalletState {
	switch s {
	case lnrpc.WalletState_NON_EXISTING:
		return domain.WalletNonExisting
	case lnrpc.WalletState_LOCKED:
		return domain.WalletLocked
	case lnrpc.WalletState_UNLOCKED:
		return domain.WalletUnlocked
	case lnrpc.WalletState_RPC_ACTIVE:
		return domain.WalletRPCActive
	case lnrpc.WalletState_SERVER_ACTIVE:
		return domain.WalletServerActive
	case lnrpc.WalletState_WAITING_TO_START:
		return domain.WalletWaitingToStart
	default:
		return domain.WalletUnknown
	}
}

func (g *Gateway) GetInfo(ctx context.Context, id domain.NodeID) (domain.NodeInfo, error) {
	resp, err := g.getInfo(ctx, id)
	if err != nil {
		return domain.NodeInfo{}, err
	}

	network := ""
	if chains := resp.GetChains(); len(chains) > 0 {
		network = chains[0].GetNetwork()
	}

	return domain.NodeInfo{
		Alias:               resp.GetAlias(),
		Color:               resp.GetColor(),
		NumPendingChannels:  resp.GetNumPendingChannels(),
		NumActiveChannels:   resp.GetNumActiveChannels(),
		NumInactiveChannels: resp.GetNumInactiveChannels(),
		NumPeers:            resp.GetNumPeers(),
		BlockHeight:         resp.GetBlockHeight(),
		Network:             network,
		URIs:                resp.GetUris(),
		SyncedToChain:       resp.GetSyncedToChain(),
		SyncedToGraph:       resp.GetSyncedToGraph(),
	}, nil
}

func (g *Gateway) getInfo(ctx context.Context, id domain.NodeID) (*lnrpc.GetInfoResponse, error) {
	client, err := g.open(id)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	resp, err := client.GetLightningClient().GetInfo(ctx, &lnrpc.GetInfoRequest{})
	if err != nil {
		return nil, exception.NewError(fmt.Sprintf("Unable to getInfo of Node[%s]", id), err)
	}
	return resp, nil
}

func (g *Gateway) UpdateAppearance(ctx context.Context, id domain.NodeID, alias, color string) error {
	client, err := g.open(id)
	if err != nil {
		return err
	}
	defer client.Close()

	if _, err = client.GetLightningClient().SetAlias(ctx, &lnrpc.SetAliasRequest{Alias: alias}); err != nil {
		return exception.NewError("cannot change the alias", err)
	}
	if _, err = client.GetLightningClient().SetColor(ctx, &lnrpc.SetColorRequest{Color: color}); err != nil {
		return exception.NewError("cannot change the color", err)
	}
	return nil
}

// ---------------- ChannelGateway ----------------

func (g *Gateway) AddPeer(ctx context.Context, id domain.NodeID, peer domain.PeerAddress) error {
	client, err := g.open(id)
	if err != nil {
		return err
	}
	defer client.Close()

	_, err = client.GetLightningClient().ConnectPeer(ctx, &lnrpc.ConnectPeerRequest{
		Addr: &lnrpc.LightningAddress{Pubkey: peer.PubKey, Host: peer.Host},
		Perm: true,
	})
	if err != nil {
		return exception.NewError("Error on adding a peer", err)
	}
	return nil
}

func (g *Gateway) OpenChannel(ctx context.Context, id domain.NodeID, peerPubKeyHex string, amountSat int64) error {
	client, err := g.open(id)
	if err != nil {
		return err
	}
	defer client.Close()

	pubkeyBytes, err := hex.DecodeString(peerPubKeyHex)
	if err != nil {
		return exception.NewError("Error on decode pubkeyHex", err)
	}

	// TODO @GoodToHave use OpenChannelSync return value for UI notification piping
	_, err = client.GetLightningClient().OpenChannelSync(ctx, &lnrpc.OpenChannelRequest{
		NodePubkey:         pubkeyBytes,
		LocalFundingAmount: amountSat,
		SatPerVbyte:        1, // fee on chain
		UseBaseFee:         true,
		BaseFee:            1000, // 1000 = 1 msat * 1000 = 1 sat
		UseFeeRate:         true,
		FeeRate:            1000, // nb sat by 1 000 000 of sat
	})
	if err != nil {
		return exception.NewError("Error on opening a channel", err)
	}
	return nil
}

// ---------------- PaymentGateway ----------------

// invoiceExpirySeconds is the lifetime of the invoices created by the playground (5 min).
const invoiceExpirySeconds = int64(300)

func (g *Gateway) CreateInvoice(ctx context.Context, id domain.NodeID, memo string, amountSat int64) (string, error) {
	client, err := g.open(id)
	if err != nil {
		return "", err
	}
	defer client.Close()

	invoice, err := client.GetLightningClient().AddInvoice(ctx, &lnrpc.Invoice{Expiry: invoiceExpirySeconds, Memo: memo, Value: amountSat})
	if err != nil {
		return "", exception.NewError("Error on creating invoice", err)
	}
	return invoice.PaymentRequest, nil
}

func (g *Gateway) PayInvoice(ctx context.Context, id domain.NodeID, paymentRequest string) (port.PaymentProgress, error) {
	client, err := g.open(id)
	if err != nil {
		return nil, err
	}

	feeLimitMsat := estimateFeeLimitMsat(ctx, client.GetRouterClient(), paymentRequest)

	// The payment outlives the request that started it: do not tie the stream to the request cancellation.
	stream, err := client.GetRouterClient().SendPaymentV2(context.WithoutCancel(ctx), &routerrpc.SendPaymentRequest{
		PaymentRequest: paymentRequest,
		FeeLimitMsat:   feeLimitMsat,
	})
	if err != nil {
		client.Close()
		return nil, exception.NewError("Error on sending payment", err)
	}
	return &paymentProgress{stream: stream, client: client}, nil
}

// estimateFeeLimitMsat calls EstimateRouteFee and applies a margin/fallback
func estimateFeeLimitMsat(ctx context.Context, client routerrpc.RouterClient, paymentRequest string) int64 {
	const fallbackMsat = int64(100000) // 100 sat

	feeResp, err := client.EstimateRouteFee(ctx, &routerrpc.RouteFeeRequest{PaymentRequest: paymentRequest})
	if err != nil {
		fmt.Println("EstimateRouteFee failed, using default fee limit (100 sat):", err)
		return fallbackMsat
	}

	// add a margin (20%) to the estimated routing fee
	feeLimitMsat := feeResp.RoutingFeeMsat + (feeResp.RoutingFeeMsat / 5)
	if feeLimitMsat == 0 {
		return fallbackMsat
	}
	return feeLimitMsat
}

// paymentProgress adapts the LND payment stream to port.PaymentProgress.
type paymentProgress struct {
	stream grpc.ServerStreamingClient[lnrpc.Payment]
	client GrpcClient
}

func (p *paymentProgress) Recv() (domain.PaymentUpdate, error) {
	payment, err := p.stream.Recv()
	if err != nil {
		return domain.PaymentUpdate{}, err // io.EOF when finished
	}
	return domain.PaymentUpdate{AmountSat: payment.GetValueSat(), Status: toPaymentStatus(payment.GetStatus())}, nil
}

func (p *paymentProgress) Close() error { return p.client.Close() }

var _ port.PaymentProgress = (*paymentProgress)(nil)

func toPaymentStatus(s lnrpc.Payment_PaymentStatus) domain.PaymentStatus {
	switch s {
	case lnrpc.Payment_INITIATED:
		return domain.PaymentInitiated
	case lnrpc.Payment_IN_FLIGHT:
		return domain.PaymentInFlight
	case lnrpc.Payment_SUCCEEDED:
		return domain.PaymentSucceeded
	case lnrpc.Payment_FAILED:
		return domain.PaymentFailed
	default:
		return domain.PaymentUnknown
	}
}
