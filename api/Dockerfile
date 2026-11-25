### Utiliser comme ceci pour générer l'image
### docker build -t lightning-playground-api:latest -f api/Dockerfile .

ARG ALPINE_TAG=3.22
ARG GO_TAG=1.25.0

# 1. Source Go l'application api
FROM golang:$GO_TAG AS compiler
WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod,id=gomod,sharing=locked \
	go mod download
COPY *.go ./
COPY api api/
# compilation statique (dépendances des libs C)
RUN --mount=type=cache,target=/root/.cache/go-build,id=gobuild-api,sharing=locked \
	--mount=type=cache,target=/go/pkg/mod,id=gomod,sharing=locked \
	CGO_ENABLED=0 \
	go build -o app-api ./api/cmd

# 2. Construction du dossier /app avec le binaire + ressources
FROM scratch AS builder
WORKDIR /app
COPY --from=compiler /app/app-api ./
COPY .env ./
COPY migrations migrations/

# 3. Construction de l'image finale (alpine ou scratch)

FROM scratch AS api-scratch
WORKDIR /app
COPY --from=builder /app ./
ENTRYPOINT ["/app/app-api"]

FROM alpine:$ALPINE_TAG AS api-alpine
WORKDIR /app
COPY --from=builder /app ./
ENTRYPOINT ["/app/app-api"]
