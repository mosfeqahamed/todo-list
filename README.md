# Todo List API

A simple Go REST API with user registration/login (JWT) and todo CRUD, backed by PostgreSQL.

## Run

```bash
cp .env.example .env      # then edit the values; JWT_SECRET must be 32+ chars (openssl rand -hex 32)
docker compose up --build
```

The API listens on `http://localhost:8080`. Tables are created automatically on startup.

## Endpoints

| Method | Path          | Auth | Body                                              |
|--------|---------------|------|---------------------------------------------------|
| POST   | `/register`   | No   | `{"email": "...", "password": "..."}` (8-72 chars) |
| POST   | `/login`      | No   | `{"email": "...", "password": "..."}` → `{"token": "..."}` |
| GET    | `/todos`      | Yes  | –                                                 |
| POST   | `/todos`      | Yes  | `{"title": "...", "description": "...", "completed": false}` |
| GET    | `/todos/{id}` | Yes  | –                                                 |
| PUT    | `/todos/{id}` | Yes  | `{"title": "...", "description": "...", "completed": true}` (full replace) |
| DELETE | `/todos/{id}` | Yes  | –                                                 |

Authenticated requests need the header `Authorization: Bearer <token>`. Tokens expire after 24 hours.

## Example

```bash
curl -X POST localhost:8080/register -d '{"email":"user@example.com","password":"password123"}'
TOKEN=$(curl -s -X POST localhost:8080/login -d '{"email":"user@example.com","password":"password123"}' | jq -r .token)

curl -X POST localhost:8080/todos -H "Authorization: Bearer $TOKEN" -d '{"title":"Buy milk"}'
curl localhost:8080/todos -H "Authorization: Bearer $TOKEN"
curl -X PUT localhost:8080/todos/1 -H "Authorization: Bearer $TOKEN" -d '{"title":"Buy milk","completed":true}'
curl -X DELETE localhost:8080/todos/1 -H "Authorization: Bearer $TOKEN"
```

## Project layout

```
cmd/api/main.go              entry point, routes, graceful shutdown
internal/config              env var loading
internal/db                  connection pool + embedded schema.sql
internal/models              User and Todo structs
internal/handlers            auth and todo HTTP handlers
internal/middleware          JWT auth middleware
```
