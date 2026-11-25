package domain

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	ErrInvalidChannelPoint = errors.New("invalid channel point")
	ErrChannelNotFound     = errors.New("channel not found")
	ErrChannelNotClosable  = errors.New("channel cannot be closed")
	ErrChannelPeerOffline  = errors.New("channel peer is offline")
)

// ChannelPoint is the funding outpoint of a channel: "<txid hex>:<output index>".
type ChannelPoint struct {
	FundingTxID string
	OutputIndex uint32
}

// ParseChannelPoint parses "<64 hex>:<uint32>"; the txid is lower-cased.
func ParseChannelPoint(raw string) (ChannelPoint, error) {
	txid, index, found := strings.Cut(raw, ":")
	if !found || len(txid) != 64 {
		return ChannelPoint{}, ErrInvalidChannelPoint
	}
	if _, err := hex.DecodeString(txid); err != nil {
		return ChannelPoint{}, ErrInvalidChannelPoint
	}
	n, err := strconv.ParseUint(index, 10, 32)
	if err != nil {
		return ChannelPoint{}, ErrInvalidChannelPoint
	}
	return ChannelPoint{FundingTxID: strings.ToLower(txid), OutputIndex: uint32(n)}, nil
}

func (p ChannelPoint) String() string {
	return fmt.Sprintf("%s:%d", p.FundingTxID, p.OutputIndex)
}

type ChannelStatus string

const (
	ChannelActive       ChannelStatus = "active"
	ChannelInactive     ChannelStatus = "inactive"
	ChannelPendingOpen  ChannelStatus = "pending_open"
	ChannelClosing      ChannelStatus = "closing"
	ChannelForceClosing ChannelStatus = "force_closing"
)

type Channel struct {
	Point            ChannelPoint
	RemotePubKey     string
	CapacitySat      int64
	LocalBalanceSat  int64
	RemoteBalanceSat int64
	Status           ChannelStatus
}

// IsClosable tells whether a close can be requested (open channels only).
func (c Channel) IsClosable() bool {
	return c.Status == ChannelActive || c.Status == ChannelInactive
}
