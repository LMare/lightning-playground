# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A personal playground: a Go + HTMX web app that drives LND (Lightning) nodes on a Bitcoin **simnet** (btcd), deployable via docker compose or Kubernetes (kind + Helm). Module path: `github.com/Lmare/lightning-playground` (Go 1.25). README/docs are partly in French.

Branch `refacto/hexagonal` migrates the backend from layered ("clean") architecture to hexagonal (ports & adapters). `master` still has the old layout (`backend/handler`, `backend/service`, ...); this file describes the hexagonal layout.

## Commands

```bash
go build ./... && go vet ./...          # CI compile + vet
go test ./...                           # all tests (unit + integration)
go test ./backend/internal/integration -run TestName -v   # single test
go test -coverpkg=./... -coverprofile=cover.out ./... -count=1   # CI coverage run (Codecov; ignores *.pb.go and testsupport)

go run ./backend/cmd      # run backend locally (needs Postgres, see below; migrations are read from ./migrations)
go run ./cmd/frontend

docker buildx bake                 # build all images (docker-bake.hcl)
docker compose up -d               # frontend on :3000
make all                           # Kubernetes: bake, kind cluster, load images, postgres (helm), deploy, hosts, monitoring
make refresh-new-version           # bump version.txt (sed across yaml/env/bake files), rebuild app images, reload, redeploy
make check-deps                    # verify docker/kind/kubectl/buildx/helm
```

LND wallets must be created manually on first launch (`lncli --network=simnet create` in each lnd container/pod); see README.md.

## Architecture

Entrypoints: `backend/cmd` (backend binary: `main.go` + `migration.go`), and `cmd/{frontend,sidecar-backend,sidecar-btcd,sidecar-lnd}` (each its own image; sidecars live in `sidecar/` and handle file/Kubernetes concerns such as macaroon/cert sharing). `config.go` at the repo root is the shared env-driven `Config` (`.env`).

Backend (`backend/internal/`) is hexagonal, split in `business/` and `infra/` (rule: `business` never imports `infra`, `bootstrap`, `testsupport`, the LND protobuf package or root config; enforced by `archtest/architecture_test.go`):
- `business/<context>/` with `domain/`, `port/` (outbound interfaces), `application/` (use cases, depend only on ports). Two contexts, independent from each other:
  - `lightning` — domain: `NodeID` (validated), `NodeSummary`, `NodeInfo`, `PeerAddress`, `PaymentUpdate`, business errors (no json tags). Ports: `NodeRepository`, `NodeInspector`, `ChannelGateway`, `PaymentGateway`, `PaymentProgress`, `PaymentNotifier`. Services: `NodeService`, `ChannelService`.
  - `user` — demo user listing (`UserRepository.FindAll`).
- `infra/` adapters, one folder per technology:
  - `grpc/lnd` — `Gateway` implements the 3 LND ports; `GrpcClientFactory`/`ConnectionResolver` build the connection from config. LND protobuf types never leave this package.
  - `postgres` — `NewPostgresDB` + user repository. `filesystem` — node repository (lists node dirs under `NodeStorage`).
  - `sse` — SSE hub (sessions) + `PaymentNotifier`.
  - `http/{common,lightning,user}` — handlers, DTOs (`dto.go`, own the json tags) and HTMX templates. Each package embeds its own `templates/` with `go:embed` and parses them once; `common` holds the shared helpers (`Fail`, `HTMXResponse`, success/error toasts), probe/version/stream handlers.
- `bootstrap/` — manual composition root (`InitApp`: DB → repos → factories → services → handlers → router). Takes repos/factories as inputs so tests inject mocks.
- `shared/exception` — app error types. `backend/gRPC/` — vendored LND `.proto` + generated stubs (regenerate with `protoc --go_out=. --go-grpc_out=.`; don't hand-edit `.pb.go`).
- `testsupport/` + `integration/` — `NewTestHarness` assembles the real router and services with mock repos and a mock gRPC factory.

Errors: `exception.NewError(msg, cause)` returns an `*AppError` carrying file:line and the cause chain (works with `errors.Is/As`; `common.LogException` prints the chain). Business validation errors are plain sentinels in `domain` (`errors.Is`).

Open items: `infra/sse` uses a single hard-coded session id (`"uniqueSession"`).

Testing: unit tests per layer (domain, application with mocked ports, `infra/grpc/lnd` with `testsupport.GrpcClientFactoryMock`) plus `integration/` tests through the real router (`NewTestHarness` exposes `MockUserRepo`, `MockNodeRepo`, `GrpcFactory.Lightning/Router` testify mocks of the LND clients). Prefer integration tests that cross the most layers. `go test -race` currently reports a pre-existing data race in `infra/sse` (`TestSessionStart`).

## Database (PostgreSQL)

Only the `users` table exists (demo data): `migrations/0001_create_user_table.{up,down}.sql` (pgcrypto, `id UUID` default `gen_random_uuid()`, `nom`, `prenom`, `age`, `email`, 3 seed rows). Migrations use golang-migrate and run **at backend startup** (`backend/cmd/migration.go`, source `file://migrations`, so the working dir must contain `migrations/`; the backend Dockerfile copies it). Connection string: `POSTGRES_DSN` (default `postgres://user:password@localhost:5432/mydb?sslmode=disable`; `.env` points to the in-cluster Bitnami service `postgres-postgresql.lightning-playground.svc.cluster.local`, db `appdb`, user `appuser`). Postgres is deployed only on Kubernetes via `make deploy-postgres` (Helm); `docker-compose.yml` has its own `postgres` service (image `POSTGRES_TAG` from `.env`, same `appuser`/`apppassword`/`appdb` credentials, `POSTGRES_DSN` overridden on the backend service).

## Docs

`docs-schema/` holds the human-written architecture docs: `BACKEND_ARCHITECTURE.md` (hexagonal layout, routes, flows; up to date, keep it in sync with the code), ARCHITECTURE.md and DETAILED_FLOWS.md (system overview; backend parts updated), KUBERNETES_ARCHITECTURE.md (not reviewed against the manifests), plus Mermaid sources. `HowTo.md` has misc setup notes (gRPC client generation, btcd image, lncli).

Other infra: `docker-compose.yml`, `docker-bake.hcl`, `kubernetes/` (manifests + Helm monitoring chart, Prometheus/Grafana). `docker/` is gitignored (ELT work lives on `feature/elt-pipeline`). Version is single-sourced in `version.txt`.
