### Utiliser comme ceci pour générer l'image
### docker build -t lightning-test-backend:latest -f backend/Dockerfile .

# 1. Build de l'application backend
FROM golang:1.25.4-alpine3.22 as Builder
WORKDIR /app
COPY *.go go.* ./
COPY cmd/backend cmd/backend/
COPY backend backend/
RUN CGO_ENABLED=0 go build -o app-backend ./cmd/backend

# 2. Construction de l'image finale
FROM scratch
WORKDIR /app
COPY --from=Builder /app/app-backend ./
COPY .env nodes.yaml ./
COPY backend/templates ./backend/templates/
ENTRYPOINT ["/app/app-backend"]
