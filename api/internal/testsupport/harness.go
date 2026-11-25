package testsupport

import (
	"net/http"

	config "github.com/Lmare/lightning-playground"
	"github.com/Lmare/lightning-playground/api/internal/bootstrap"
)

// TestHarness holds components for testing the application, such as the router and mock repositories.
type TestHarness struct {
	Router       http.Handler // application router
	OpsRouter    http.Handler // probes and metrics
	MockUserRepo *MockUserRepository
	MockNodeRepo *MockNodeRepository
	GrpcFactory  *GrpcClientFactoryMock
}

// NewTestHarness initializes the test harness with default mock implementations.
func NewTestHarness() *TestHarness {
	// random test configuration
	cfg := &config.Config{
		ProjectPath: "testdata",
		NodeStorage: "testdata/nodes/",
		LndNetwork:  "simnet",
		LndDomain:   "lnd.test",
		Version:     "1.0.0",
	}

	// mock repositories
	mockuserRepo := &MockUserRepository{}
	mockNodeRepo := &MockNodeRepository{}

	// mock factories
	grpcClientFactoryMock := NewGrpcClientFactoryMock()

	// wiring with mocks
	repos := &bootstrap.Repositories{
		User: mockuserRepo,
		Node: mockNodeRepo,
	}

	factories := &bootstrap.Factories{
		GrpcClientFactory: grpcClientFactoryMock,
	}

	services := bootstrap.InitServices(repos, factories, cfg)
	handlers := bootstrap.InitHandlers(services, cfg)
	routers := bootstrap.InitRouters(handlers)

	return &TestHarness{
		Router:       routers.App,
		OpsRouter:    routers.Ops,
		MockUserRepo: mockuserRepo,
		MockNodeRepo: mockNodeRepo,
		GrpcFactory:  grpcClientFactoryMock,
	}
}
