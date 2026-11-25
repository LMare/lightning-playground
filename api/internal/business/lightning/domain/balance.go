package domain

// OnChainBalance is the wallet (UTXO) balance of the default account.
type OnChainBalance struct {
	ConfirmedSat      int64 // >= 1 conf
	UnconfirmedSat    int64 // 0 conf
	LockedSat         int64 // UTXOs leased for other usage
	ReservedAnchorSat int64 // reserve kept for anchor channels
}

// TotalSat is the confirmed plus unconfirmed amount. Locked and reserved are not added.
func (b OnChainBalance) TotalSat() int64 { return b.ConfirmedSat + b.UnconfirmedSat }

// ChannelBalance is the sum over the node's channels.
type ChannelBalance struct {
	LocalSat, RemoteSat                       int64 // open channels
	PendingOpenLocalSat, PendingOpenRemoteSat int64
	UnsettledLocalSat, UnsettledRemoteSat     int64 // in-flight HTLCs
}

// TotalLocalSat is what the node owns in its channels.
func (b ChannelBalance) TotalLocalSat() int64 {
	return b.LocalSat + b.UnsettledLocalSat + b.PendingOpenLocalSat
}

// Balance groups both views of a node's funds.
type Balance struct {
	OnChain  OnChainBalance
	Channels ChannelBalance
}

// TotalSat is the on-chain total plus the local funds in channels.
func (b Balance) TotalSat() int64 { return b.OnChain.TotalSat() + b.Channels.TotalLocalSat() }
