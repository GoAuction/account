FROM golang:1.25.5 AS dev

WORKDIR /app

RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN go mod download

COPY . .

CMD ["air", "-c", ".air.toml"]

FROM golang:1.25.4 AS builder

WORKDIR /src

# Download dependencies first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build the gRPC service binary
FROM builder AS builder-grpc
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /auction-grpc ./cmd/grpc/main.go

# gRPC Service Runner
FROM gcr.io/distroless/base-debian12:nonroot AS grpc

WORKDIR /app

COPY --from=builder-grpc /auction-grpc /usr/local/bin/auction-grpc

EXPOSE 9090

ENTRYPOINT ["/usr/local/bin/auction-grpc"]
