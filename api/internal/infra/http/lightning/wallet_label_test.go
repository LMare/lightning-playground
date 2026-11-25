package lightninghttp

import (
	"testing"

	"github.com/Lmare/lightning-playground/api/internal/business/lightning/domain"
	"github.com/stretchr/testify/assert"
)

func TestWalletLabel(t *testing.T) {
	tests := map[domain.WalletState]string{
		domain.WalletServerActive:   "Unlocked",
		domain.WalletRPCActive:      "Starting",
		domain.WalletUnlocked:       "Starting",
		domain.WalletWaitingToStart: "Starting",
		domain.WalletLocked:         "Locked",
		domain.WalletNonExisting:    "No wallet",
		domain.WalletUnreachable:    "Unreachable",
		domain.WalletUnknown:        "Unknown",
		domain.WalletState("weird"): "Unknown",
	}
	for state, want := range tests {
		assert.Equal(t, want, walletLabel(state), string(state))
	}
}
