package bootstrap

import (
	"database/sql"

	config "github.com/Lmare/lightning-playground"
	lightningapp "github.com/Lmare/lightning-playground/backend/internal/business/lightning/application"
	"github.com/Lmare/lightning-playground/backend/internal/business/lightning/port"
	userapp "github.com/Lmare/lightning-playground/backend/internal/business/user/application"
	userport "github.com/Lmare/lightning-playground/backend/internal/business/user/port"
	"github.com/Lmare/lightning-playground/backend/internal/infra/filesystem"
	lnd "github.com/Lmare/lightning-playground/backend/internal/infra/grpc/lnd"
	"github.com/Lmare/lightning-playground/backend/internal/infra/http/common"
	lightninghttp "github.com/Lmare/lightning-playground/backend/internal/infra/http/lightning"
	userhttp "github.com/Lmare/lightning-playground/backend/internal/infra/http/user"
	"github.com/Lmare/lightning-playground/backend/internal/infra/http/web"
	"github.com/Lmare/lightning-playground/backend/internal/infra/postgres"
	userpostgres "github.com/Lmare/lightning-playground/backend/internal/infra/postgres"
	sse "github.com/Lmare/lightning-playground/backend/internal/infra/sse"
	exception "github.com/Lmare/lightning-playground/backend/internal/shared/exception"
)

// Repositories groups outbound ports wired for the application.
type Repositories struct {
	User userport.UserRepository
	Node port.NodeRepository
}

// Factories groups technical factories (e.g. LND gRPC).
type Factories struct {
	GrpcClientFactory lnd.GrpcClientFactory
}

type services struct {
	user    *userapp.UserService
	node    *lightningapp.NodeService
	channel *lightningapp.ChannelService
}

type handlers struct {
	user        *userhttp.Handler
	lightning   *lightninghttp.Handler
	probe       *common.ProbeHandler
	version     *common.VersionHandler
	streamEvent *common.StreamHandler
	web         *web.Handler
}

// ------------------- Initialization functions -------------------

// InitDB opens the Postgres connection.
func InitDB(cfg *config.Config) (*sql.DB, error) {
	return postgres.NewPostgresDB(cfg.DSN)
}

// InitRepositories builds persistence adapters.
func InitRepositories(db *sql.DB, conf *config.Config) *Repositories {
	return &Repositories{
		User: userpostgres.NewPostgresUserRepository(db),
		Node: filesystem.NewNodeRepositoryFileSystem(conf),
	}
}

// InitFactories builds shared factories.
func InitFactories() *Factories {
	return &Factories{
		GrpcClientFactory: lnd.NewGrpcClientFactory(),
	}
}

// InitServices wires the application layer and LND services.
func InitServices(r *Repositories, f *Factories, conf *config.Config) *services {
	lnd := lnd.NewGateway(f.GrpcClientFactory, lnd.NewConfigConnectionResolver(conf))
	return &services{
		user:    userapp.NewUserService(r.User),
		node:    lightningapp.NewNodeService(r.Node, lnd, lnd),
		channel: lightningapp.NewChannelService(lnd, lnd, sse.NewPaymentNotifier()),
	}
}

// InitHandlers builds HTTP adapters.
func InitHandlers(s *services, conf *config.Config) *handlers {
	return &handlers{
		user:        userhttp.NewHandler(s.user),
		lightning:   lightninghttp.NewHandler(s.node, s.channel),
		probe:       common.NewProbeHandler(),
		version:     common.NewVersionHandler(conf.Version),
		streamEvent: common.NewStreamHandler(),
		web:         web.NewHandler(),
	}
}

// InitApp is the root composition entrypoint for the backend binary.
func InitApp(cfg *config.Config) (*sql.DB, Routers, error) {
	db, err := InitDB(cfg)
	if err != nil {
		return nil, Routers{}, exception.NewError("Failed to initialize database", err)
	}
	repos := InitRepositories(db, cfg)
	factories := InitFactories()
	services := InitServices(repos, factories, cfg)
	handlers := InitHandlers(services, cfg)
	return db, InitRouters(handlers), nil
}
