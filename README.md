# LONG Hub

## Deploy

### Docker

```sh
$ cp .env.template .env
$ vim .env
$ docker compose up -d
```

Frontend port is mapped to localhost:5173 and backend port is mapped to localhost:8000.

### Manual

When deploying manually or setting up for development, `.env` files are required respectively for frontend and backend, as when in Docker it is loaded by Docker Compose from the root directory.

Requirements:
- Go >= 1.26
- Node.js >= 25 with corepack enabled
- [goose](https://github.com/pressly/goose)
- PostgreSQL


Extra environment variables in `backend/.env`:
```
GOOSE_DRIVER=postgres
GOOSE_DBSTRING=postgres://user:password@host/db  # replace this
GOOSE_MIGRATION_DIR=./db/migrations
```

```sh
$ git clone https://github.com/MoveToEx/longhub.git
$ cd longhub
$ cd frontend
$ cp .env.template .env
$ vim .env
$ yarn
$ yarn dev

### (in another terminal)
$ cd backend
$ cp .env.template .env
$ vim .env
$ goose up
$ go mod download
$ go build
$ ./long
```
