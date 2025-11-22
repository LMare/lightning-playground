### Utiliser comme ceci pour générer l'image
### docker build -t lightning-playground-frontend:latest -f frontend/Dockerfile .

# 1. Build de l'application backend
FROM golang:1.25.4-alpine3.22 as Builder
WORKDIR /app
COPY *.go go.* ./
COPY cmd/frontend cmd/frontend/
RUN CGO_ENABLED=0 go build -o app-frontend ./cmd/frontend

# 2. Construction de l'image finale
FROM scratch
WORKDIR /app
COPY --from=Builder /app/app-frontend ./
COPY .env ./
COPY frontend ./frontend/
ENTRYPOINT ["/app/app-frontend"]
