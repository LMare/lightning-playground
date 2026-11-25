package domain

import "strings"

// PeerAddress is the address of a Lightning peer: "<pubkey>@<host:port>".
type PeerAddress struct {
	PubKey string
	Host   string
}

// ParsePeerAddress parses a "<pubkey>@<host:port>" URI.
func ParsePeerAddress(uri string) (PeerAddress, error) {
	pubKey, host, found := strings.Cut(uri, "@")
	if !found || pubKey == "" || host == "" {
		return PeerAddress{}, ErrInvalidPeerAddress
	}
	return PeerAddress{PubKey: pubKey, Host: host}, nil
}
