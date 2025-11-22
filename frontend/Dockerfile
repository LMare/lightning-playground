### Utiliser comme ceci pour générer l'image
### docker build -t lightning-playground-frontend:latest -f frontend/Dockerfile .

# 1. Build de l'application frontend
FROM golang:1.25.4-alpine3.22 as Compiler
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY cmd/frontend cmd/frontend/
RUN CGO_ENABLED=0 go build -o app-frontend ./cmd/frontend

# 2. Construction du dossier /app avec le binaire + ressources
FROM scratch as Builder
WORKDIR /app
COPY --from=Compiler /app/app-frontend ./
COPY .env ./
COPY frontend ./frontend/

# 3. Construction de l'image finale
FROM scratch
WORKDIR /app
COPY --from=Builder /app ./
ENTRYPOINT ["/app/app-frontend"]
