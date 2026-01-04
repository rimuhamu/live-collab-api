# Live Collaboration API - Setup Guide

## Prerequisites

- Go 1.24+
- Docker and Docker Compose

## Setup Steps

### 1. Start the database
```bash
docker-compose up -d
```

### 2. Create environment file

Create `.env` in the project root:
```env
DATABASE_URL=
JWT_SECRET=
FRONTEND_URL=
ALLOWED_ORIGINS=
REDIS_URL=
```

### 3. Install dependencies
```bash
go mod download
```

### 4. Run the server
```bash
go run cmd/server/main.go
```

Server runs at `http://localhost:8080`

## Testing the API

### View API Documentation

Open in browser:
```
http://localhost:8080/swagger/index.html
```

### Running Tests

Run all tests:
```bash
go test ./...
```

Run with verbose output:
```bash
go test -v ./...
```

Run tests with coverage:
```bash
go test ./... -cover
```

Test specific package:
```bash
go test ./internal/auth -v
go test ./internal/documents -v
```

## Stopping the Server

- Stop server: `Ctrl+C`
- Stop database: `docker-compose down`
- Stop and remove data: `docker-compose down -v`