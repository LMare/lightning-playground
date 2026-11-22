package application

import (
	"context"
	"log"

	"github.com/Lmare/lightning-playground/backend/internal/business/lightning/domain"
	"github.com/Lmare/lightning-playground/backend/internal/business/lightning/port"
)

func NewNodeService(repo port.NodeRepository, inspector port.NodeInspector) *NodeService {
	return &NodeService{repo: repo, inspector: inspector}
}

// NodeService is the use case set about the nodes themselves.
type NodeService struct {
	repo      port.NodeRepository
	inspector port.NodeInspector
}

// ListNodes returns the summary of every reachable node; unreachable nodes are skipped.
func (s *NodeService) ListNodes(ctx context.Context) ([]domain.NodeSummary, error) {
	ids, err := s.repo.GetNodesIds()
	if err != nil {
		return nil, err
	}

	nodes := make([]domain.NodeSummary, 0, len(ids))
	for _, raw := range ids {
		id, err := domain.ParseNodeID(raw)
		if err != nil {
			log.Printf("[WARN] ignoring node %q: %v", raw, err)
			continue
		}
		summary, err := s.inspector.GetSummary(ctx, id)
		if err != nil {
			log.Printf("[WARN] ignoring node %q: %v", raw, err)
			continue
		}
		nodes = append(nodes, summary)
	}
	return nodes, nil
}

// GetInfo returns the detailed state of a node.
func (s *NodeService) GetInfo(ctx context.Context, id domain.NodeID) (domain.NodeInfo, error) {
	return s.inspector.GetInfo(ctx, id)
}

// GetURI returns the first public URI of a node ("" if none).
func (s *NodeService) GetURI(ctx context.Context, id domain.NodeID) (string, error) {
	info, err := s.inspector.GetInfo(ctx, id)
	if err != nil {
		return "", err
	}
	return info.FirstURI(), nil
}

// UpdateAppearance changes the alias and the color of a node.
func (s *NodeService) UpdateAppearance(ctx context.Context, id domain.NodeID, alias, color string) error {
	return s.inspector.UpdateAppearance(ctx, id, alias, color)
}
