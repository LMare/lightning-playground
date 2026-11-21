package lightninghttp

import "github.com/Lmare/lightning-playground/backend/internal/lightning/domain"

// DTOs expose the domain over HTTP. The field names are also used by the HTML templates.

type nodeSummaryDTO struct {
	Id    string `json:"id"`
	Alias string `json:"alias"`
	Color string `json:"color"`
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
		dtos = append(dtos, nodeSummaryDTO{Id: n.ID.String(), Alias: n.Alias, Color: n.Color})
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
