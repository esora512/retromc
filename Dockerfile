FROM golang:1.25-bookworm AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags "-X main.GitCommit=$(git rev-parse --short HEAD 2>/dev/null || echo unknown) -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o retromc .

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=build /app/retromc .

EXPOSE 25565

CMD ["./retromc", "--host", "0.0.0.0"]
