# Build stage
FROM golang:1.25-alpine AS builder

# Install only what we need for BPF compilation
RUN apk add --no-cache \
    clang \
    llvm \
    build-base \
    libbpf-dev \
    linux-headers \
    make \
    git

ENV GOCACHE=/root/.cache/go-build
ENV GOMODCACHE=/go/pkg/mod

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build eBPF program
RUN make bpf

# Build Go binary with version metadata stamped via ldflags. Falls back to
# "unknown" when the build context lacks .git (e.g. CI tarballs).
RUN COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo unknown) \
    && BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
    && CGO_ENABLED=0 go build \
        -ldflags="-s -w -X github.com/zrougamed/cerberus/internal/version.Commit=${COMMIT} -X github.com/zrougamed/cerberus/internal/version.Date=${BUILD_DATE}" \
        -o cerberus cmd/cerberus/main.go

# Runtime stage
FROM alpine:latest

# Install minimal runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    iproute2

WORKDIR /app

# Copy compiled artifacts
COPY --from=builder /app/cerberus /app/cerberus
COPY --from=builder /app/build/cerberus_tc.o /app/cerberus_tc.o

# Create data directory
RUN mkdir -p /app/data

CMD ["./cerberus"]
