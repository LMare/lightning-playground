package lndgrpc

import (
	config "github.com/Lmare/lightning-playground"
	"github.com/Lmare/lightning-playground/backend/internal/lightning/domain"
)

// ConnectionResolver finds how to reach a node from its id.
type ConnectionResolver interface {
	Resolve(id domain.NodeID) NodeConnection
}

type configConnectionResolver struct {
	nodeStorage string
	network     string
	domain      string
}

// NewConfigConnectionResolver derives certificate, macaroon and address from the configuration.
func NewConfigConnectionResolver(conf *config.Config) ConnectionResolver {
	return &configConnectionResolver{nodeStorage: conf.NodeStorage, network: conf.LndNetwork, domain: conf.LndDomain}
}

func (r *configConnectionResolver) Resolve(id domain.NodeID) NodeConnection {
	return NodeConnection{
		TLSCertPath:  r.nodeStorage + string(id) + "/tls.cert",
		MacaroonPath: r.nodeStorage + string(id) + "/data/chain/bitcoin/" + r.network + "/admin.macaroon",
		Address:      string(id) + "." + r.domain + ":10009",
	}
}
