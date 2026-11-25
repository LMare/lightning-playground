package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWalletState_IsReady(t *testing.T) {
	tests := map[WalletState]bool{
		WalletNonExisting:    false,
		WalletLocked:         false,
		WalletUnlocked:       false,
		WalletRPCActive:      true,
		WalletServerActive:   true,
		WalletWaitingToStart: false,
		WalletUnknown:        false,
		WalletUnreachable:    false,
		WalletState(""):      false,
	}
	for state, want := range tests {
		assert.Equal(t, want, state.IsReady(), string(state))
	}
}
