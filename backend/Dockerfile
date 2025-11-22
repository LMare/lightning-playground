### Utiliser comme ceci pour générer l'image
### docker build -t lightning-playground-backend:latest -f backend/Dockerfile .

# 1. Build de l'application backend
FROM golang:1.25.4-alpine3.22 as Compiler
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY cmd/backend cmd/backend/
COPY backend backend/
RUN CGO_ENABLED=0 go build -o app-backend ./cmd/backend

# 2. Construction du dossier /app avec le binaire + ressources
FROM scratch as Builder
WORKDIR /app
COPY --from=Compiler /app/app-backend ./
COPY .env nodes.yaml ./
COPY backend/templates ./backend/templates/

# 3. Construction de l'image finale
FROM scratch
WORKDIR /app
COPY --from=Builder /app ./
ENTRYPOINT ["/app/app-backend"]
