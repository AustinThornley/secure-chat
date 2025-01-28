# ──────────────────────────────────────────────────────────────────────────
# 1) Build Stage: Cross-compile the server for `linux/amd64` architecture
# ──────────────────────────────────────────────────────────────────────────
FROM --platform=linux/amd64 golang:1.20-bullseye AS build

# Enable CGO and set the target OS/architecture
ENV CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=amd64

# Set the working directory
WORKDIR /app

# Install the required dependencies for go-sqlcipher
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    g++ \
    libc6-dev \
    libssl-dev \
    sqlite3 \
    libsqlite3-dev \
 && rm -rf /var/lib/apt/lists/*

# Copy the server code
COPY server/server.go ./server.go

# Initialize Go modules (if needed)
RUN go mod init ephemeral-chat || true
RUN go mod tidy || true

# Compile the Go server for Linux (amd64)
RUN go build -o server server.go

# ──────────────────────────────────────────────────────────────────────────
# 2) Runtime Stage: Add missing glibc dependencies
# ──────────────────────────────────────────────────────────────────────────
FROM --platform=linux/amd64 debian:bullseye-slim AS runtime

# Copy the compiled binary from the build stage
COPY --from=build /app/server /server

# Ensure glibc and ld-linux-x86-64.so.2 are present
RUN apt-get update && apt-get install -y --no-install-recommends \
    libc6 \
 && rm -rf /var/lib/apt/lists/*

# Expose the chat server port
EXPOSE 9000

# Start the server
ENTRYPOINT ["/server"]
