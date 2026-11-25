package domain_test

import (
	"strings"
	"testing"

	"github.com/Lmare/lightning-playground/api/internal/business/lightning/domain"
	"github.com/stretchr/testify/assert"
)

var txid64 = strings.Repeat("ab", 32)

func TestParseChannelPoint_Valid(t *testing.T) {
	for _, raw := range []string{txid64 + ":0", txid64 + ":4294967295", txid64 + ":7"} {
		p, err := domain.ParseChannelPoint(raw)
		assert.NoError(t, err, raw)
		assert.Equal(t, raw, p.String())
	}
	p, _ := domain.ParseChannelPoint(txid64 + ":4294967295")
	assert.Equal(t, txid64, p.FundingTxID)
	assert.Equal(t, uint32(4294967295), p.OutputIndex)
}

func TestParseChannelPoint_LowerCasesTxID(t *testing.T) {
	p, err := domain.ParseChannelPoint(strings.ToUpper(txid64) + ":1")
	assert.NoError(t, err)
	assert.Equal(t, txid64+":1", p.String())
}

func TestParseChannelPoint_Invalid(t *testing.T) {
	for _, raw := range []string{
		"", "nocolon", txid64, txid64[:63] + ":0", txid64 + "a:0",
		strings.Repeat("zz", 32) + ":0", txid64 + ":-1", txid64 + ":abc",
		txid64 + ":4294967296", txid64 + ":1:2", txid64 + ":", ":0",
	} {
		_, err := domain.ParseChannelPoint(raw)
		assert.ErrorIs(t, err, domain.ErrInvalidChannelPoint, raw)
	}
}

func TestChannel_IsClosable(t *testing.T) {
	cases := map[domain.ChannelStatus]bool{
		domain.ChannelActive:       true,
		domain.ChannelInactive:     true,
		domain.ChannelPendingOpen:  false,
		domain.ChannelClosing:      false,
		domain.ChannelForceClosing: false,
	}
	for st, want := range cases {
		assert.Equal(t, want, domain.Channel{Status: st}.IsClosable(), st)
	}
}
