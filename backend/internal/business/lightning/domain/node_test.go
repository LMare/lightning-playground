package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseNodeID(t *testing.T) {
	tests := []struct {
		name  string
		id    string
		valid bool
	}{
		{"letters", "node1", true},
		{"underscore", "lnd_node", true},
		{"dash", "my-node", true},
		{"all chars", "node-1_test", true},
		{"space", "node 1", false},
		{"special chars", "node@1", false},
		{"dot", "node.1", false},
		{"path traversal", "../etc", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := ParseNodeID(tt.id)
			if tt.valid {
				assert.NoError(t, err)
				assert.Equal(t, NodeID(tt.id), id)
			} else {
				assert.ErrorIs(t, err, ErrInvalidNodeID)
			}
		})
	}
}

func TestNodeInfoFirstURI(t *testing.T) {
	assert.Equal(t, "", NodeInfo{}.FirstURI())
	assert.Equal(t, "a@h:1", NodeInfo{URIs: []string{"a@h:1", "b@h:2"}}.FirstURI())
}

func TestParsePeerAddress(t *testing.T) {
	peer, err := ParsePeerAddress("02abc@lnd-1:9735")
	assert.NoError(t, err)
	assert.Equal(t, PeerAddress{PubKey: "02abc", Host: "lnd-1:9735"}, peer)

	for _, bad := range []string{"", "02abc", "@host", "02abc@"} {
		_, err := ParsePeerAddress(bad)
		assert.ErrorIs(t, err, ErrInvalidPeerAddress, bad)
	}
}

func TestPaymentUpdateString(t *testing.T) {
	assert.Equal(t, "💸 Paiement de 42 sats — statut : SUCCEEDED", PaymentUpdate{AmountSat: 42, Status: PaymentSucceeded}.String())
}
