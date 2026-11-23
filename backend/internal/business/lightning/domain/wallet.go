package domain

import "errors"

var (
	// ErrWalletPasswordRequired is returned when the wallet password is empty.
	ErrWalletPasswordRequired = errors.New("wallet password required")
	// ErrInvalidWalletPassword is returned when the node rejects the wallet password.
	ErrInvalidWalletPassword = errors.New("invalid wallet password")
	// ErrWalletNotLocked is returned when the wallet is already unlocked, starting or active.
	ErrWalletNotLocked = errors.New("wallet is not locked")
	// ErrWalletNotCreated is returned when the node has no wallet yet.
	ErrWalletNotCreated = errors.New("wallet not created")
)

// WalletState is the lifecycle state of a node's wallet, as seen by the playground.
type WalletState string

const (
	WalletNonExisting    WalletState = "non_existing"     // wallet not created yet
	WalletLocked         WalletState = "locked"           // needs the password
	WalletUnlocked       WalletState = "unlocked"         // unlocked, RPC server not ready
	WalletRPCActive      WalletState = "rpc_active"       // RPC up, server not fully ready
	WalletServerActive   WalletState = "server_active"    // fully ready
	WalletWaitingToStart WalletState = "waiting_to_start" // waiting for cluster leadership
	WalletUnknown        WalletState = "unknown"          // value LND sent that we do not know
	WalletUnreachable    WalletState = "unreachable"      // could not ask the node at all
)

// IsReady reports whether the lnrpc Lightning service can answer (GetInfo, ...).
func (s WalletState) IsReady() bool { return s == WalletRPCActive || s == WalletServerActive }
