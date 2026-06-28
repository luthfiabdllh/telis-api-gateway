# ============================================================
# Stage 1: Builder
# Build binary Go dari source code
# ============================================================
FROM golang:alpine AS builder

WORKDIR /app

# Install dependencies untuk build (gcc diperlukan beberapa CGO libs)
RUN apk add --no-cache git ca-certificates tzdata

# Copy module files dulu (layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy seluruh source code
COPY . .

# Build binary (static, tanpa CGO agar bisa jalan di alpine minimal)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/gateway ./cmd/api/main.go

# ============================================================
# Stage 2: Runtime
# Image final yang kecil dan aman
# ============================================================
FROM alpine:3.19

WORKDIR /app

# Install CA certificates dan timezone data
RUN apk add --no-cache ca-certificates tzdata

# Copy binary dari builder stage
COPY --from=builder /app/gateway .

# Copy docs swagger (jika ada)
COPY --from=builder /app/docs ./docs

EXPOSE 8000

CMD ["./gateway"]
