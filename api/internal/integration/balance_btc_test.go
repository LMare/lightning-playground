package integration

import (
	"net/http"
	"testing"

	lnrpc "github.com/Lmare/lightning-playground/api/gRPC/github.com/lightningnetwork/lnd/lnrpc"
	"github.com/Lmare/lightning-playground/api/internal/testsupport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Large amounts must show an approximate BTC value (2 decimals) next to the sat figure,
// while the sat values and data attributes stay unchanged.
func TestBalance_HTMX_ShowsBTCApproximation(t *testing.T) {
	h := testsupport.NewTestHarness()
	stateIs(h, lnrpc.WalletState_SERVER_ACTIVE)
	h.GrpcFactory.Lightning.On("WalletBalance", mock.Anything, mock.Anything).Return(&lnrpc.WalletBalanceResponse{
		ConfirmedBalance: 123456700,
	}, nil)
	h.GrpcFactory.Lightning.On("ChannelBalance", mock.Anything, mock.Anything).Return(&lnrpc.ChannelBalanceResponse{
		LocalBalance:  &lnrpc.Amount{Sat: 250000000},
		RemoteBalance: &lnrpc.Amount{Sat: 100000000},
	}, nil)

	rr := doHTMX(h, http.MethodGet, balancePath+"?id=lnd-0", nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	for _, s := range []string{
		// unchanged sat values and attributes
		`data-onchain-confirmed="123456700"`, `data-onchain-total="123456700"`,
		`data-channels-local="250000000"`, `data-channels-remote="100000000"`,
		`data-channels-total="250000000"`, `data-total="373456700"`,
		// BTC approximations: on-chain 1.23, channels local 2.50, remote 1.00, total 3.73
		"1.23", "2.50", "1.00", "3.73", "BTC",
	} {
		assert.Contains(t, body, s)
	}
}
