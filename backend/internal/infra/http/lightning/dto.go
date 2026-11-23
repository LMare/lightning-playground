package lightninghttp

import "github.com/Lmare/lightning-playground/backend/internal/business/lightning/domain"

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
