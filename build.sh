go build -ldflags "-X main.GitCommit=$(git rev-parse --short HEAD) -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
