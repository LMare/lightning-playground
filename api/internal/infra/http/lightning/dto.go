package lightninghttp

import "github.com/Lmare/lightning-playground/api/internal/business/lightning/domain"

// DTOs expose the domain over HTTP. The field names are also used by the HTML templates.

type nodeSummaryDTO struct {
	Id          string `json:"id"`
	Alias       string `json:"alias"`
	Color       string `json:"color"`
	WalletState string `json:"walletState"`
	WalletLabel string `json:"walletLabel"`
}

type nodeInfoDTO struct {
	Alias               string   `json:"alias"`
	Color               string   `json:"color"`
	NumPendingChannels  uint32   `json:"numPendingChannels"`
	NumActiveChannels   uint32   `json:"numActiveChannels"`
	NumInactiveChannels uint32   `json:"numInactiveChannels"`
	NumPeers            uint32   `json:"numPeers"`
	BlockHeight         uint32   `json:"blockHeight"`
	Network             string   `json:"network"`
	Uris                []string `json:"uris"`
	SyncedToChain       bool     `json:"syncedToChain"`
	SyncedToGraph       bool     `json:"syncedToGraph"`
}

func toNodeSummaryDTOs(nodes []domain.NodeSummary) []nodeSummaryDTO {
	dtos := make([]nodeSummaryDTO, 0, len(nodes))
	for _, n := range nodes {
		dtos = append(dtos, nodeSummaryDTO{Id: n.ID.String(), Alias: n.Alias, Color: n.Color,
			WalletState: string(n.WalletState), WalletLabel: walletLabel(n.WalletState)})
	}
	return dtos
}

func toNodeInfoDTO(n domain.NodeInfo) nodeInfoDTO {
	return nodeInfoDTO{
		Alias:               n.Alias,
		Color:               n.Color,
		NumPendingChannels:  n.NumPendingChannels,
		NumActiveChannels:   n.NumActiveChannels,
		NumInactiveChannels: n.NumInactiveChannels,
		NumPeers:            n.NumPeers,
		BlockHeight:         n.BlockHeight,
		Network:             n.Network,
		Uris:                n.URIs,
		SyncedToChain:       n.SyncedToChain,
		SyncedToGraph:       n.SyncedToGraph,
	}
}

// walletLabel groups the wallet states into the labels shown in the node list.
func walletLabel(s domain.WalletState) string {
	switch s {
	case domain.WalletServerActive:
		return "Unlocked"
	case domain.WalletRPCActive, domain.WalletUnlocked, domain.WalletWaitingToStart:
		return "Starting"
	case domain.WalletLocked:
		return "Locked"
	case domain.WalletNonExisting:
		return "No wallet"
	case domain.WalletUnreachable:
		return "Unreachable"
	default:
		return "Unknown"
	}
}

type channelDTO struct {
	ChannelPoint     string `json:"channelPoint"`
	RemotePubKey     string `json:"remotePubKey"`
	CapacitySat      int64  `json:"capacitySat"`
	LocalBalanceSat  int64  `json:"localBalanceSat"`
	RemoteBalanceSat int64  `json:"remoteBalanceSat"`
	Status           string `json:"status"`
	StatusLabel      string `json:"statusLabel"`
	Closable         bool   `json:"closable"`
}

// channelsView is the model of the HTMX channels template.
type channelsView struct {
	NodeID   string
	Channels []channelDTO
}

type closeChannelDTO struct {
	ClosingTxID string `json:"closingTxId"`
}

func toChannelDTOs(channels []domain.Channel) []channelDTO {
	dtos := make([]channelDTO, 0, len(channels))
	for _, c := range channels {
		dtos = append(dtos, channelDTO{
			ChannelPoint:     c.Point.String(),
			RemotePubKey:     c.RemotePubKey,
			CapacitySat:      c.CapacitySat,
			LocalBalanceSat:  c.LocalBalanceSat,
			RemoteBalanceSat: c.RemoteBalanceSat,
			Status:           string(c.Status),
			StatusLabel:      channelStatusLabel(c.Status),
			Closable:         c.IsClosable(),
		})
	}
	return dtos
}

func channelStatusLabel(s domain.ChannelStatus) string {
	switch s {
	case domain.ChannelActive:
		return "Active"
	case domain.ChannelInactive:
		return "Inactive"
	case domain.ChannelPendingOpen:
		return "Opening"
	case domain.ChannelClosing:
		return "Closing"
	case domain.ChannelForceClosing:
		return "Force closing"
	default:
		return "Unknown"
	}
}

type onChainBalanceDTO struct {
	ConfirmedSat      int64 `json:"confirmedSat"`
	UnconfirmedSat    int64 `json:"unconfirmedSat"`
	TotalSat          int64 `json:"totalSat"`
	LockedSat         int64 `json:"lockedSat"`
	ReservedAnchorSat int64 `json:"reservedAnchorSat"`
}

type channelBalanceDTO struct {
	LocalSat             int64 `json:"localSat"`
	RemoteSat            int64 `json:"remoteSat"`
	PendingOpenLocalSat  int64 `json:"pendingOpenLocalSat"`
	PendingOpenRemoteSat int64 `json:"pendingOpenRemoteSat"`
	UnsettledLocalSat    int64 `json:"unsettledLocalSat"`
	UnsettledRemoteSat   int64 `json:"unsettledRemoteSat"`
	TotalLocalSat        int64 `json:"totalLocalSat"`
}

type balanceDTO struct {
	OnChain  onChainBalanceDTO `json:"onChain"`
	Channels channelBalanceDTO `json:"channels"`
	TotalSat int64             `json:"totalSat"`
}

func toBalanceDTO(b domain.Balance) balanceDTO {
	return balanceDTO{
		OnChain: onChainBalanceDTO{
			ConfirmedSat:      b.OnChain.ConfirmedSat,
			UnconfirmedSat:    b.OnChain.UnconfirmedSat,
			TotalSat:          b.OnChain.TotalSat(),
			LockedSat:         b.OnChain.LockedSat,
			ReservedAnchorSat: b.OnChain.ReservedAnchorSat,
		},
		Channels: channelBalanceDTO{
			LocalSat:             b.Channels.LocalSat,
			RemoteSat:            b.Channels.RemoteSat,
			PendingOpenLocalSat:  b.Channels.PendingOpenLocalSat,
			PendingOpenRemoteSat: b.Channels.PendingOpenRemoteSat,
			UnsettledLocalSat:    b.Channels.UnsettledLocalSat,
			UnsettledRemoteSat:   b.Channels.UnsettledRemoteSat,
			TotalLocalSat:        b.Channels.TotalLocalSat(),
		},
		TotalSat: b.TotalSat(),
	}
}

// nodesView is the model of the HTMX nodes template: the rows and the node whose panel is open.
type nodesView struct {
	Selected string
	Nodes    []nodeSummaryDTO
}

// nodeInfoView is the model of the HTMX nodeInfo template.
type nodeInfoView struct {
	NodeID string
	Info   nodeInfoDTO
}
