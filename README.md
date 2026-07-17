# Tanok Tech Web API

Backend API en Go para [tanok-tech.com](https://tanok-tech.com).

## Stack

- Go 1.25 + Gin framework
- PostgreSQL (pgx v5)
- Redis 7 (rate limiting + cache)
- Cloudflare R2 (image storage)
- MiniMax image-01 (cover generation)
- Docker + Docker Compose (Dockploy)

## Desarrollo local

```bash
go mod download
go run ./cmd/api
```

Variables mínimas para arrancar:

```env
DATABASE_URL=postgres://...
JWT_SECRET=...
REDIS_URL=redis://localhost:6379
```

## Compilar

```bash
CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/api ./cmd/api
```

## Docker

```bash
docker compose up -d
```

La imagen se construye en multi-stage (golang:1.25 → alpine:3.21). El binario pesa ~10MB.

## Endpoints

| Método | Ruta | Auth |
|---|---|---|
| `GET` | `/healthz` | — |
| `POST` | `/api/auth/login` | — |
| `POST` | `/api/auth/logout` | — |
| `GET` | `/api/auth/me` | JWT cookie |
| `GET` | `/api/posts` | — |
| `GET` | `/api/posts/:slug` | — |
| `GET` | `/api/posts/:slug/sibling` | — |
| `POST` | `/api/leads/capture` | rate limit |
| `GET` | `/api/leads` | JWT cookie |
| `GET` | `/api/leads/:id` | JWT cookie |
| `POST` | `/api/insights/track` | rate limit |
| `GET` | `/api/insights/stats` | JWT cookie |
| `GET` | `/api/insights/events` | JWT cookie |
| `GET` | `/api/insights/top-elements` | JWT cookie |
| `GET` | `/api/insights/engagement` | JWT cookie |
| `GET` | `/api/insights/heatmap` | JWT cookie |
| `GET` | `/api/insights/traffic-sources` | JWT cookie |
| `POST` | `/api/insights/analyze` | JWT cookie |
| `POST` | `/api/send-web-questionnaire` | rate limit |
| `POST` | `/api/cron/daily-posts` | CRON_SECRET |
| `POST` | `/api/images/generate-cover` | JWT cookie |

## Variables de entorno

Copiar `.env.example` a `.env` y completar:

```env
PORT=8080
GIN_MODE=release

# Base de datos
DATABASE_URL=postgres://user:pass@host:5432/tanok?sslmode=disable

# Redis
REDIS_URL=redis://redis-cache:6379

# Auth
JWT_SECRET=<random 32+ chars>
API_TOKEN=<random 32+ chars>
CRON_SECRET=<random 32+ chars>

# Dominios
BASE_URL=https://tanok-web-api.tanok-tech.com
COOKIE_DOMAIN=tanok-tech.com
CORS_ALLOWED_ORIGINS=https://tanok-tech.com,https://www.tanok-tech.com

# Email (Resend)
RESEND_API_KEY=re_...

# AI
DEEPSEEK_API_KEY=sk-...
DEEPSEEK_MODEL=deepseek-v4-flash
MINIMAX_API_KEY=...
BRAVE_API_KEY=...

# Images (Cloudflare R2)
R2_ACCESS_KEY_ID=...
R2_SECRET_ACCESS_KEY=...
R2_BUCKET_NAME=tanok-web-images
R2_PUBLIC_URL=https://pub-xxx.r2.dev
R2_ENDPOINT=https://xxx.r2.cloudflarestorage.com
R2_REGION=auto

# Cron
DAILY_POSTS_AUTHOR_ID=<user-id-from-db>
AGENT_PIPELINE_ENABLED=true
```

## Cron job en Dockploy

Crear un cron job que apunte a:

```
POST https://tanok-web-api.tanok-tech.com/api/cron/daily-posts
Header: Authorization: Bearer <CRON_SECRET>
Schedule: 0 */6 * * *
```
