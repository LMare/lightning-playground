package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Lmare/lightning-playground/backend/internal/lightning/application"
	"github.com/Lmare/lightning-playground/backend/internal/lightning/domain"
	"github.com/Lmare/lightning-playground/backend/internal/testsupport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestListNodes_SkipsInvalidAndUnreachableNodes(t *testing.T) {
	repo := &testsupport.MockNodeRepository{}
	inspector := &testsupport.MockNodeInspector{}
	repo.On("GetNodesIds").Return([]string{"lnd-0", "bad id", "lnd-1", "lnd-2"}, nil)
	inspector.On("GetSummary", mock.Anything, domain.NodeID("lnd-0")).Return(domain.NodeSummary{ID: "lnd-0", Alias: "alice", Color: "#fff"}, nil)
	inspector.On("GetSummary", mock.Anything, domain.NodeID("lnd-1")).Return(domain.NodeSummary{}, errors.New("unreachable"))
	inspector.On("GetSummary", mock.Anything, domain.NodeID("lnd-2")).Return(domain.NodeSummary{ID: "lnd-2", Alias: "carol"}, nil)

	nodes, err := application.NewNodeService(repo, inspector).ListNodes(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, []domain.NodeSummary{{ID: "lnd-0", Alias: "alice", Color: "#fff"}, {ID: "lnd-2", Alias: "carol"}}, nodes)
}

func TestListNodes_RepositoryError(t *testing.T) {
	repo := &testsupport.MockNodeRepository{}
	repo.On("GetNodesIds").Return([]string(nil), errors.New("no storage"))

	_, err := application.NewNodeService(repo, &testsupport.MockNodeInspector{}).ListNodes(context.Background())

	assert.Error(t, err)
}

func TestGetURI(t *testing.T) {
	inspector := &testsupport.MockNodeInspector{}
	inspector.On("GetInfo", mock.Anything, domain.NodeID("a")).Return(domain.NodeInfo{URIs: []string{"k@h:1"}}, nil)
	inspector.On("GetInfo", mock.Anything, domain.NodeID("b")).Return(domain.NodeInfo{}, nil)
	svc := application.NewNodeService(&testsupport.MockNodeRepository{}, inspector)

	uri, err := svc.GetURI(context.Background(), "a")
	assert.NoError(t, err)
	assert.Equal(t, "k@h:1", uri)

	uri, err = svc.GetURI(context.Background(), "b")
	assert.NoError(t, err)
	assert.Equal(t, "", uri)
}

func newChannelService() (*application.ChannelService, *testsupport.MockChannelGateway, *testsupport.MockPaymentGateway, *testsupport.MockPaymentNotifier) {
	channels, payments, notifier := &testsupport.MockChannelGateway{}, &testsupport.MockPaymentGateway{}, &testsupport.MockPaymentNotifier{}
	return application.NewChannelService(channels, payments, notifier), channels, payments, notifier
}

func TestAddPeer(t *testing.T) {
	svc, channels, _, _ := newChannelService()
	channels.On("AddPeer", mock.Anything, domain.NodeID("n"), domain.PeerAddress{PubKey: "02ab", Host: "h:9735"}).Return(nil)

	assert.NoError(t, svc.AddPeer(context.Background(), "n", "02ab@h:9735"))
	channels.AssertExpectations(t)
}

func TestAddPeer_InvalidAddressNeverReachesTheGateway(t *testing.T) {
	svc, channels, _, _ := newChannelService()

	err := svc.AddPeer(context.Background(), "n", "no-at-sign")

	assert.ErrorIs(t, err, domain.ErrInvalidPeerAddress)
	channels.AssertNotCalled(t, "AddPeer", mock.Anything, mock.Anything, mock.Anything)
}

func TestAmountMustBePositive(t *testing.T) {
	svc, channels, payments, _ := newChannelService()

	assert.ErrorIs(t, svc.OpenChannel(context.Background(), "n", "02ab", 0), domain.ErrInvalidAmount)
	_, err := svc.CreateInvoice(context.Background(), "n", "memo", -1)
	assert.ErrorIs(t, err, domain.ErrInvalidAmount)
	channels.AssertNotCalled(t, "OpenChannel", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	payments.AssertNotCalled(t, "CreateInvoice", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

type fakeProgress struct{}

func (fakeProgress) Recv() (domain.PaymentUpdate, error) { return domain.PaymentUpdate{}, nil }
func (fakeProgress) Close() error                        { return nil }

func TestPay_HandsTheProgressToTheNotifier(t *testing.T) {
	svc, _, payments, notifier := newChannelService()
	progress := fakeProgress{}
	payments.On("PayInvoice", mock.Anything, domain.NodeID("n"), "lnbc1").Return(progress, nil)
	notifier.On("Watch", progress).Return()

	assert.NoError(t, svc.Pay(context.Background(), "n", "lnbc1"))
	notifier.AssertExpectations(t)
}

func TestPay_GatewayErrorDoesNotNotify(t *testing.T) {
	svc, _, payments, notifier := newChannelService()
	payments.On("PayInvoice", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("boom"))

	assert.Error(t, svc.Pay(context.Background(), "n", "lnbc1"))
	notifier.AssertNotCalled(t, "Watch", mock.Anything)
}
