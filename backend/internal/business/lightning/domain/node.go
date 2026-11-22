package domain

import (
	"errors"
	"regexp"
)

var (
	// ErrInvalidNodeID is returned when a node identifier has an unsupported format.
	ErrInvalidNodeID = errors.New("invalid node id")
	// ErrInvalidPeerAddress is returned when a peer address is not "<pubkey>@<host>".
	ErrInvalidPeerAddress = errors.New("invalid peer address")
	// ErrInvalidAmount is returned when an amount is not strictly positive.
	ErrInvalidAmount = errors.New("invalid amount")
)

var nodeIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// NodeID identifies a Lightning node managed by the playground (ex: "lnd-0").
type NodeID string

// ParseNodeID validates a raw identifier. Only letters, digits, '_' and '-' are accepted,
// which also keeps the id safe to use in file paths and host names.
func ParseNodeID(raw string) (NodeID, error) {
	if !nodeIDPattern.MatchString(raw) {
		return "", ErrInvalidNodeID
	}
	return NodeID(raw), nil
}

func (id NodeID) String() string { return string(id) }

// NodeSummary is the minimal description of a node, used in lists.
type NodeSummary struct {
	ID    NodeID
	Alias string
	Color string
	// WalletState is the wallet lifecycle state of the node.
	WalletState WalletState
}

// NodeInfo is the detailed state of a node.
type NodeInfo struct {
	Alias               string
	Color               string
	NumPendingChannels  uint32
	NumActiveChannels   uint32
	NumInactiveChannels uint32
	NumPeers            uint32
	BlockHeight         uint32
	Network             string
	URIs                []string
	SyncedToChain       bool
	SyncedToGraph       bool
}

// FirstURI returns the first public URI of the node, or "" when it has none.
func (n NodeInfo) FirstURI() string {
	if len(n.URIs) == 0 {
		return ""
	}
	return n.URIs[0]
}
