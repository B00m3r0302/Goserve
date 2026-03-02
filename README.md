# Goserve

A RESTful HTTP API server built in Go, modeled after a micro-blogging platform ("Chirpy"). It handles user registration, JWT-based authentication, short-post creation (chirps), and a premium tier (Chirpy Red) managed via webhook.

## Features

- User registration and login with Argon2id password hashing
- JWT access tokens (1-hour expiry) and long-lived refresh tokens (60-day expiry)
- Create, read, and delete chirps (140-character limit)
- Profanity filtering on chirp content
- Filter and sort chirps by author or timestamp
- Chirpy Red premium tier upgradeable via Polka webhook
- Admin metrics and dev-mode database reset endpoints
- Static file serving at `/app/`
- PostgreSQL backend with sqlc-generated query layer

## Tech Stack

| Component     | Library / Tool                         |
|---------------|----------------------------------------|
| Language      | Go (stdlib `net/http`)                 |
| Database      | PostgreSQL                             |
| Query gen     | [sqlc](https://sqlc.dev/)              |
| Migrations    | [goose](https://github.com/pressly/goose) |
| JWT           | `github.com/golang-jwt/jwt/v5`         |
| Password hash | `github.com/alexedwards/argon2id`      |
| UUIDs         | `github.com/google/uuid`               |
| Env vars      | `github.com/joho/godotenv`             |

## Prerequisites

- Go 1.21+
- PostgreSQL
- `sqlc` (for regenerating DB layer)
- `goose` (for running migrations)

## Setup

1. **Clone the repository**

   ```bash
   git clone https://github.com/B00m3r0302/Goserve.git
   cd Goserve
   ```

2. **Create a `.env` file** in the project root:

   ```env
   DB_URL=postgres://user:password@localhost:5432/chirpy?sslmode=disable
   SECRET_KEY=your-jwt-secret
   POLKA_KEY=your-polka-api-key
   PLATFORM=dev   # set to "dev" to enable /admin/reset
   ```

3. **Run database migrations**

   ```bash
   goose -dir sql/schema postgres "$DB_URL" up
   ```

4. **Build and run**

   ```bash
   go build -o Goserve .
   ./Goserve
   ```

   The server listens on `:8080`.

## API Reference

### Health

| Method | Path            | Auth | Description        |
|--------|-----------------|------|--------------------|
| GET    | `/api/healthz`  | None | Returns `200 OK`   |

### Users

| Method | Path         | Auth          | Description                    |
|--------|--------------|---------------|--------------------------------|
| POST   | `/api/users` | None          | Register a new user            |
| POST   | `/api/login` | None          | Login and receive tokens       |
| PUT    | `/api/users` | Bearer JWT    | Update email and/or password   |

**Register / Login request body:**
```json
{
  "email": "user@example.com",
  "password": "secret"
}
```

**Login response:**
```json
{
  "id": "uuid",
  "email": "user@example.com",
  "is_chirpy_red": false,
  "token": "<jwt>",
  "refresh_token": "<refresh_token>",
  "created_at": "...",
  "updated_at": "..."
}
```

### Tokens

| Method | Path           | Auth         | Description                             |
|--------|----------------|--------------|-----------------------------------------|
| POST   | `/api/refresh` | Bearer token | Exchange refresh token for new JWT      |
| POST   | `/api/revoke`  | Bearer token | Revoke a refresh token                  |

### Chirps

| Method | Path                      | Auth       | Description                                  |
|--------|---------------------------|------------|----------------------------------------------|
| POST   | `/api/chirps`             | Bearer JWT | Create a chirp (max 140 chars)               |
| GET    | `/api/chirps`             | None       | List all chirps                              |
| GET    | `/api/chirps/{chirpID}`   | None       | Get a single chirp by ID                     |
| DELETE | `/api/chirps/{chirpID}`   | Bearer JWT | Delete a chirp (owner only)                  |

**Query parameters for `GET /api/chirps`:**

| Param       | Values          | Default | Description                  |
|-------------|-----------------|---------|------------------------------|
| `author_id` | UUID            | —       | Filter chirps by user ID     |
| `sort`      | `asc` / `desc`  | `asc`   | Sort by `created_at`         |

**Create chirp request body:**
```json
{
  "body": "Hello, Chirpy!"
}
```

Profanity filter: the words `kerfuffle`, `sharbert`, and `fornax` (case-insensitive) are replaced with `****`.

### Webhooks

| Method | Path                    | Auth   | Description                          |
|--------|-------------------------|--------|--------------------------------------|
| POST   | `/api/polka/webhooks`   | ApiKey | Upgrade a user to Chirpy Red         |

The `Authorization` header must be `ApiKey <POLKA_KEY>`. Only `user.upgraded` events are acted upon.

**Request body:**
```json
{
  "event": "user.upgraded",
  "data": {
    "user_id": "uuid"
  }
}
```

### Admin

| Method | Path              | Auth | Description                                        |
|--------|-------------------|------|----------------------------------------------------|
| GET    | `/admin/metrics`  | None | HTML page with fileserver hit count                |
| POST   | `/admin/reset`    | None | Reset users table (dev mode only — requires `PLATFORM=dev`) |

## Project Structure

```
Goserve/
├── main.go                  # Server setup and route registration
├── structs.go               # Shared types (apiConfig, User)
├── chirpEndpoints.go        # Chirp CRUD handlers
├── userEndpoints.go         # User and webhook handlers
├── serverEndpoints.go       # Health, metrics, refresh, revoke handlers
├── internal/
│   ├── auth/
│   │   ├── tokenFunctions.go    # JWT creation/validation, bearer extraction, refresh token gen
│   │   └── passwordFunctions.go # Argon2id hashing and API key extraction
│   └── database/                # sqlc-generated database layer
└── sql/
    ├── schema/                  # goose migration files
    └── queries/                 # sqlc SQL queries
```

## Running Tests

```bash
go test ./...
```

## License

MIT