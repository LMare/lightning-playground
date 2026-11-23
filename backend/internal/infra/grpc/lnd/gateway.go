package lnd

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	lnrpc "github.com/Lmare/lightning-playground/backend/gRPC/github.com/lightningnetwork/lnd/lnrpc"
	routerrpc "github.com/Lmare/lightning-playground/backend/gRPC/github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"github.com/Lmare/lightning-playground/backend/internal/business/lightning/domain"
	"github.com/Lmare/lightning-playground/backend/internal/business/lightning/port"
	exception "github.com/Lmare/lightning-playground/backend/internal/shared/exception"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// Gateway is the outbound adapter talking to LND nodes over gRPC.
// It implements the NodeInspector, WalletGateway, ChannelGateway and PaymentGateway ports.
type Gateway struct {
	factory  GrpcClientFactory
	resolver ConnectionResolver
}

var (
	_ port.NodeInspector  = (*Gateway)(nil)
	_ port.ChannelGateway = (*Gateway)(nil)
	_ port.PaymentGateway = (*Gateway)(nil)
	_ port.WalletGateway  = (*Gateway)(nil)
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

// ---------------- WalletGateway ----------------

// Unlock decrypts the wallet of a node. The password is never put in an error or a log.
func (g *Gateway) Unlock(ctx context.Context, id domain.NodeID, password []byte) error {
	client, err := g.open(id)
	if err != nil {
		return err
	}
	defer client.Close()

	if _, err = client.GetWalletUnlockerClient().UnlockWallet(ctx, &lnrpc.UnlockWalletRequest{WalletPassword: password}); err != nil {
		return toUnlockError(id, err)
	}
	return nil
}

// toUnlockError maps the LND error text (lnd v0.20) to the domain errors.
func toUnlockError(id domain.NodeID, err error) error {
	msg := status.Convert(err).Message()
	switch {
	case strings.Contains(msg, "invalid passphrase"):
		return exception.NewError(fmt.Sprintf("Wrong wallet password for Node[%s]", id), domain.ErrInvalidWalletPassword)
	case strings.Contains(msg, "already unlocked"):
		return exception.NewError(fmt.Sprintf("Wallet of Node[%s] is not locked", id), domain.ErrWalletNotLocked)
	case strings.Contains(msg, "wallet not found"):
		return exception.NewError(fmt.Sprintf("No wallet on Node[%s]", id), domain.ErrWalletNotCreated)
	default:
		return exception.NewError(fmt.Sprintf("Unable to unlock wallet of Node[%s]", id), err)
	}
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

func (g *Gateway) ListChannels(ctx context.Context, id domain.NodeID) ([]domain.Channel, error) {
	client, err := g.open(id)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	wrap := func(err error) error {
		return exception.NewError(fmt.Sprintf("Unable to list channels of Node[%s]", id), err)
	}
	open, err := client.GetLightningClient().ListChannels(ctx, &lnrpc.ListChannelsRequest{})
	if err != nil {
		return nil, wrap(err)
	}
	pending, err := client.GetLightningClient().PendingChannels(ctx, &lnrpc.PendingChannelsRequest{})
	if err != nil {
		return nil, wrap(err)
	}

	channels := make([]domain.Channel, 0, len(open.GetChannels()))
	add := func(rawPoint, remote string, capacity, local, remoteBal int64, status domain.ChannelStatus) error {
		point, err := domain.ParseChannelPoint(rawPoint)
		if err != nil {
			return wrap(fmt.Errorf("malformed channel point %q: %w", rawPoint, err))
		}
		channels = append(channels, domain.Channel{Point: point, RemotePubKey: remote, CapacitySat: capacity,
			LocalBalanceSat: local, RemoteBalanceSat: remoteBal, Status: status})
		return nil
	}
	for _, c := range open.GetChannels() {
		status := domain.ChannelInactive
		if c.GetActive() {
			status = domain.ChannelActive
		}
		if err := add(c.GetChannelPoint(), c.GetRemotePubkey(), c.GetCapacity(), c.GetLocalBalance(), c.GetRemoteBalance(), status); err != nil {
			return nil, err
		}
	}
	addPending := func(c *lnrpc.PendingChannelsResponse_PendingChannel, status domain.ChannelStatus) error {
		return add(c.GetChannelPoint(), c.GetRemoteNodePub(), c.GetCapacity(), c.GetLocalBalance(), c.GetRemoteBalance(), status)
	}
	for _, c := range pending.GetPendingOpenChannels() {
		if err := addPending(c.GetChannel(), domain.ChannelPendingOpen); err != nil {
			return nil, err
		}
	}
	for _, c := range pending.GetWaitingCloseChannels() {
		if err := addPending(c.GetChannel(), domain.ChannelClosing); err != nil {
			return nil, err
		}
	}
	for _, c := range pending.GetPendingForceClosingChannels() {
		if err := addPending(c.GetChannel(), domain.ChannelForceClosing); err != nil {
			return nil, err
		}
	}
	return channels, nil
}

// CloseChannel requests the close and returns the closing txid from the first update of the stream.
// LND goes on with the close after the stream is cancelled.
func (g *Gateway) CloseChannel(ctx context.Context, id domain.NodeID, point domain.ChannelPoint, force bool) (string, error) {
	client, err := g.open(id)
	if err != nil {
		return "", err
	}
	defer client.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stream, err := client.GetLightningClient().CloseChannel(ctx, &lnrpc.CloseChannelRequest{
		ChannelPoint: &lnrpc.ChannelPoint{
			FundingTxid: &lnrpc.ChannelPoint_FundingTxidStr{FundingTxidStr: point.FundingTxID},
			OutputIndex: point.OutputIndex,
		},
		Force: force,
	})
	if err != nil {
		return "", exception.NewError("Error on closing the channel", err)
	}
	update, err := stream.Recv()
	if err != nil {
		return "", exception.NewError("Error on closing the channel", err)
	}
	switch u := update.GetUpdate().(type) {
	case *lnrpc.CloseStatusUpdate_ClosePending:
		return txidToString(u.ClosePending.GetTxid()), nil
	case *lnrpc.CloseStatusUpdate_ChanClose:
		return txidToString(u.ChanClose.GetClosingTxid()), nil
	default:
		return "", exception.NewError("Error on closing the channel", fmt.Errorf("unexpected close update %T", u))
	}
}

// txidToString converts raw hash bytes (internal order) to the usual reversed hex txid.
func txidToString(raw []byte) string {
	reversed := make([]byte, len(raw))
	for i, b := range raw {
		reversed[len(raw)-1-i] = b
	}
	return hex.EncodeToString(reversed)
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

// GetBalance reads the on-chain (WalletBalance) and channel (ChannelBalance) balances of a node.
func (g *Gateway) GetBalance(ctx context.Context, id domain.NodeID) (domain.Balance, error) {
	client, err := g.open(id)
	if err != nil {
		return domain.Balance{}, err
	}
	defer client.Close()

	lightning := client.GetLightningClient()
	wallet, err := lightning.WalletBalance(ctx, &lnrpc.WalletBalanceRequest{})
	if err != nil {
		return domain.Balance{}, exception.NewError(fmt.Sprintf("Unable to get balance of Node[%s]", id), err)
	}
	channels, err := lightning.ChannelBalance(ctx, &lnrpc.ChannelBalanceRequest{})
	if err != nil {
		return domain.Balance{}, exception.NewError(fmt.Sprintf("Unable to get balance of Node[%s]", id), err)
	}
	return domain.Balance{
		OnChain: domain.OnChainBalance{
			ConfirmedSat:      wallet.GetConfirmedBalance(),
			UnconfirmedSat:    wallet.GetUnconfirmedBalance(),
			LockedSat:         wallet.GetLockedBalance(),
			ReservedAnchorSat: wallet.GetReservedBalanceAnchorChan(),
		},
		Channels: domain.ChannelBalance{
			LocalSat:             satOf(channels.GetLocalBalance()),
			RemoteSat:            satOf(channels.GetRemoteBalance()),
			UnsettledLocalSat:    satOf(channels.GetUnsettledLocalBalance()),
			UnsettledRemoteSat:   satOf(channels.GetUnsettledRemoteBalance()),
			PendingOpenLocalSat:  satOf(channels.GetPendingOpenLocalBalance()),
			PendingOpenRemoteSat: satOf(channels.GetPendingOpenRemoteBalance()),
		},
	}, nil
}

// satOf returns the sat value of an LND amount; nil gives 0.
func satOf(a *lnrpc.Amount) int64 { return int64(a.GetSat()) }
