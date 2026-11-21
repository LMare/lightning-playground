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
docker compose up -d               # frontend on :3000 (no Postgres service in the compose file)
make all                           # Kubernetes: bake, kind cluster, load images, postgres (helm), deploy, hosts, monitoring
make refresh-new-version           # bump version.txt (sed across yaml/env/bake files), rebuild app images, reload, redeploy
make check-deps                    # verify docker/kind/kubectl/buildx/helm
```

LND wallets must be created manually on first launch (`lncli --network=simnet create` in each lnd container/pod); see README.md.

## Architecture

Entrypoints: `backend/cmd` (backend binary: `main.go` + `migration.go`), and `cmd/{frontend,sidecar-backend,sidecar-btcd,sidecar-lnd}` (each its own image; sidecars live in `sidecar/` and handle file/Kubernetes concerns such as macaroon/cert sharing). `config.go` at the repo root is the shared env-driven `Config` (`.env`).

Backend is organised per bounded context under `backend/internal/`, each with `domain/`, `port/` (interfaces), `application/` (services) and `adapter/` (implementations):
- `lightning/` — LND nodes. `adapter/filesystem` (NodeRepository: lists node dirs under `NodeStorage`), `adapter/grpc` (LND gRPC: `GrpcClientFactory`, `LightningInfoService`, `ChannelService`), `adapter/http` (HTMX handlers), `application/node_service.go` (builds cert/macaroon/URL for a node id).
- `user/` — demo user CRUD. `adapter/postgres`, `adapter/http`, `application`, `port`, `domain`.
- `bootstrap/` — manual composition root (`InitApp`: DB → repos → factories → services → handlers → router). Takes repos/factories as inputs so tests inject mocks.
- `httpapi/` — shared HTTP helpers (HTMX/JSON responses, `Fail`), probe/version/stream handlers.
- `platform/` — technical infra: `postgres` (sql.DB), `stream` (SSE sessions).
- `shared/exception` — app error types. `gRPC/` — vendored LND `.proto` + generated stubs (regenerate with `protoc --go_out=. --go-grpc_out=.`; don't hand-edit `.pb.go`).
- `testsupport/` + `integration/` — `NewTestHarness` assembles the real router with mock repos and a mock gRPC factory (`GrpcClientFactoryMock`, defined in `adapter/grpc/grpc_client_factory.go`).

Known hexagonal impurities (in progress): `lightning/adapter/http` handlers depend on the concrete gRPC adapter services (`lndgrpc.LightningInfoService`/`ChannelService`) rather than on ports; there is no application/port layer for LND info/channel operations; the `GrpcClientFactory` interface has unexported methods; `NodeService` takes the whole `config.Config`.

## Database (PostgreSQL)

Only the `users` table exists (demo data): `migrations/0001_create_user_table.{up,down}.sql` (pgcrypto, `id UUID` default `gen_random_uuid()`, `nom`, `prenom`, `age`, `email`, 3 seed rows). Migrations use golang-migrate and run **at backend startup** (`backend/cmd/migration.go`, source `file://migrations`, so the working dir must contain `migrations/`; the backend Dockerfile copies it). Connection string: `POSTGRES_DSN` (default `postgres://user:password@localhost:5432/mydb?sslmode=disable`; `.env` points to the in-cluster Bitnami service `postgres-postgresql.lightning-playground.svc.cluster.local`, db `appdb`, user `appuser`). Postgres is deployed only on Kubernetes via `make deploy-postgres` (Helm); `docker-compose.yml` has no Postgres service.

## Docs

`docs-schema/` holds the human-written architecture docs (ARCHITECTURE.md, DETAILED_FLOWS.md, KUBERNETES_ARCHITECTURE.md + Mermaid sources). They predate the hexagonal refactor. `HowTo.md` has misc setup notes (gRPC client generation, btcd image, lncli).

Other infra: `docker-compose.yml`, `docker-bake.hcl`, `kubernetes/` (manifests + Helm monitoring chart, Prometheus/Grafana). `docker/` is gitignored (ELT work lives on `feature/elt-pipeline`). Version is single-sourced in `version.txt`.
