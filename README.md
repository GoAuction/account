# Account Service

User profile management service. Provides REST API, gRPC service for profile queries, and event-driven profile creation.

## Architecture

| Component | Binary | Role |
|-----------|--------|------|
| **API** | `cmd/api/` | REST endpoints for profirele management |
| **gRPC** | `cmd/grpc/` | Profile queries for other services |
| **Worker** | `cmd/worker/` | Event consumption for profile creation |

## Quick Start

```bash
# Start shared infrastructure first
docker compose -f infra/docker/docker-compose.yaml up -d

# Development mode (hot reload)
docker compose --profile dev up

# Production mode
docker compose --profile prod up
```

### Local Development

```bash
# API
POSTGRES_HOST=localhost \
POSTGRES_PORT=5435 \
POSTGRES_DATABASE=account \
POSTGRES_USERNAME=postgres \
POSTGRES_PASSWORD=postgres \
PORT=8080 \
go run cmd/api/main.go

# gRPC
POSTGRES_HOST=localhost \
POSTGRES_PORT=5435 \
GRPC_PORT=9091 \
go run cmd/grpc/main.go

# Worker
POSTGRES_HOST=localhost \
POSTGRES_PORT=5435 \
RABBITMQ_URL=amqp://guest:guest@localhost:5672/ \
go run cmd/worker/main.go
```

## API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/user-profiles/:id` | X-User-ID | Get user profile |
| PUT | `/api/v1/user-profiles/:id` | X-User-ID | Update user profile |

### Profile Fields

- `name` - Display name
- `phone` - Phone number
- `country` - Country
- `city` - City
- `description` - Bio/description
- `photo` - Profile photo URL
- `phone_verified` - Phone verification status
- `email_verified` - Email verification status
- `business` - Business account flag
- `verified` - Verified account flag

## gRPC Service

**Port:** `9091` (configurable via `GRPC_PORT`)

```protobuf
service AccountService {
  rpc GetProfile(GetProfileRequest) returns (GetProfileResponse);
}
```

Used by the bid service to sync user profile data.

## Events

### Consumed (Worker ← RabbitMQ)

| Event | Exchange | Action |
|-------|----------|--------|
| `identity.user.registered.v1` | `auction.identity` | Creates initial profile |

When a user registers via Identity service, Account worker automatically creates their profile.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP API port |
| `HOST_PORT` | `8084` | External mapped port |
| `GRPC_PORT` | `9091` | gRPC server port |
| `POSTGRES_HOST` | `account-postgres` | Database host |
| `POSTGRES_PORT` | `5432` | Database port |
| `POSTGRES_DATABASE` | `account` | Database name |
| `POSTGRES_USERNAME` | `postgres` | Database user |
| `POSTGRES_PASSWORD` | `postgres` | Database password |
| `RABBITMQ_URL` | - | RabbitMQ connection |
| `SERVICE_NAME` | `account` | Service identifier |

## Database

Uses GORM with auto-migration (no manual migrations needed).

**Entity:** `Profile` - user profile data with verification flags

## Project Structure

```
account/
├── cmd/
│   ├── api/              # REST API
│   ├── grpc/             # gRPC server
│   └── worker/           # Event consumer
├── app/                  # HTTP handlers
├── domain/               # Profile entity
├── internal/
│   ├── middleware/       # Security headers
│   └── consumers/        # Event handlers
├── infra/
│   ├── grpc/             # gRPC server implementation
│   └── rabbitmq/         # Publisher + consumer
├── proto/                # Protocol buffers
└── pkg/                  # Shared utilities
```

## Service Interactions

```
Identity Service                Account Service
     │                               │
     │ identity.user.registered.v1   │
     ├──────────────────────────────►│ Worker creates profile
     │                               │
                                     │
Bid Service                          │
     │        GetProfile (gRPC)      │
     ├──────────────────────────────►│ Returns profile data
     │                               │
```
