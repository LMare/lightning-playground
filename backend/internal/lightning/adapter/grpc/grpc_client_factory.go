package lndgrpc

import (
	"context"
	"crypto/x509"
	"fmt"
	"os"

	lnrpc "github.com/Lmare/lightning-playground/backend/gRPC/github.com/lightningnetwork/lnd/lnrpc"
	routerrpc "github.com/Lmare/lightning-playground/backend/gRPC/github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	exception "github.com/Lmare/lightning-playground/backend/internal/shared/exception"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// NodeConnection holds what is needed to open a secure gRPC connection to a node.
type NodeConnection struct {
	TLSCertPath  string
	MacaroonPath string
	Address      string
}

// GrpcClientFactory opens gRPC clients to an LND node.
type GrpcClientFactory interface {
	NewClient(conn NodeConnection) (GrpcClient, error)
}

// GrpcClient gives access to the LND services; Close releases the connection.
type GrpcClient interface {
	GetLightningClient() lnrpc.LightningClient
	GetRouterClient() routerrpc.RouterClient
	Close() error
}

//----------------- GrpcClient Factory Impl-----------------

type grpcClientFactoryImpl struct {
}

// NewClient opens a real gRPC connection.
func (f *grpcClientFactoryImpl) NewClient(dataClient NodeConnection) (GrpcClient, error) {
	conn, err := f.createGrpcClientConn(dataClient)
	if err != nil {
		err := exception.NewError("Error creating gRPC communication channel", err, exception.NewExampleError)
		return nil, err
	}

	return &grpcClientImpl{
		lightningClient: lnrpc.NewLightningClient(conn),
		routerClient:    routerrpc.NewRouterClient(conn),
		conn:            conn,
	}, nil
}

// createGrpcClientConn create a secure gRPC connection
func (f *grpcClientFactoryImpl) createGrpcClientConn(dataClient NodeConnection) (*grpc.ClientConn, error) {

	// Charger le certificat TLS
	cert, err := os.ReadFile(dataClient.TLSCertPath)
	if err != nil {
		err := exception.NewError(fmt.Sprintf("Erreur lecture TLS cert : %s", dataClient.TLSCertPath), err, exception.NewExampleError)
		return nil, err
	}
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(cert)
	creds := credentials.NewClientTLSFromCert(certPool, "")

	// Load macaroon
	macaroonBytes, err := os.ReadFile(dataClient.MacaroonPath)
	if err != nil {
		err := exception.NewError(fmt.Sprintf("Error reading macaroon: %s", dataClient.MacaroonPath), err, exception.NewExampleError)
		return nil, err
	}
	macaroonHex := fmt.Sprintf("%x", macaroonBytes)

	// Créer un dial gRPC sécurisé
	return grpc.NewClient(
		dataClient.Address,
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(newMacaroonCreds(macaroonHex)),
	)
}

// constructor for the factory
func NewGrpcClientFactory() GrpcClientFactory {
	return &grpcClientFactoryImpl{}
}

//----------------- Macaroon Credentials -----------------

// macaroonCreds attaches the macaroon to outgoing gRPC metadata.
type macaroonCreds struct {
	macaroon string
}

func (m macaroonCreds) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{"macaroon": m.macaroon}, nil
}

func (m macaroonCreds) RequireTransportSecurity() bool {
	return true
}

func newMacaroonCreds(macaroonHex string) credentials.PerRPCCredentials {
	return macaroonCreds{macaroon: macaroonHex}
}

//----------------- GrpcClient Implementation -----------------

// grpcClientImpl wraps gRPC clients for dialing and tests.
type grpcClientImpl struct {
	lightningClient lnrpc.LightningClient
	routerClient    routerrpc.RouterClient
	conn            *grpc.ClientConn
}

// GetLightningClient returns the Lightning RPC client.
func (f *grpcClientImpl) GetLightningClient() lnrpc.LightningClient {
	return f.lightningClient
}

// GetRouterClient returns the Router RPC client.
func (f *grpcClientImpl) GetRouterClient() routerrpc.RouterClient {
	return f.routerClient
}

// Close the gRPC connection
func (f *grpcClientImpl) Close() error {
	if f.conn != nil {
		return f.conn.Close()
	}
	return nil
}
