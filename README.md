# iam-go

🌐 **English** | [Bahasa Indonesia](README.id.md)

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![gRPC](https://img.shields.io/badge/gRPC-go--grpc-244c5a)](https://grpc.io)
[![PRs welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

**Identity & Access Management** — Auth + User microservices with **granular
RBAC**, built in **Go**. This is the **platform/umbrella** repo: it orchestrates
the independently-deployed services and holds the deployment, docs and API
collections. Sibling Rust implementation: [iam-rust](https://github.com/malvinpratama/iam-rust).

> Stack: **Go · Gin** (REST gateway) · **gRPC** (inter-service) · **NATS JetStream**
> (async events) · **PostgreSQL** (one DB per service) · **sqlc** · **JWT**
> (access + refresh, revocable).

## Repositories

Each service is its own repo — built, versioned and deployed independently;
shared code lives in dedicated module repos.

| Repo | Role |
|---|---|
| [iam-go-gateway](https://github.com/malvinpratama/iam-go-gateway) | REST→gRPC API gateway, per-route authorization |
| [iam-go-auth](https://github.com/malvinpratama/iam-go-auth) | Auth + RBAC gRPC service (owns `auth_db`, publishes events) |
| [iam-go-user](https://github.com/malvinpratama/iam-go-user) | Profile gRPC service (owns `user_db`, consumes events) |
| [iam-go-contracts](https://github.com/malvinpratama/iam-go-contracts) | Shared `.proto` + generated gRPC stubs |
| [iam-go-libs](https://github.com/malvinpratama/iam-go-libs) | Shared libraries (config, db, jwt, NATS, …) |
| **iam-go** (this repo) | Platform: compose · k8s · docs · collections · smoke |

## Features

- 🔐 **Auth**: register, login, JWT access + refresh tokens with **rotation** and **revocation**.
- 👤 **Users**: profile CRUD + paginated search, via a dedicated service.
- 🛡️ **Granular RBAC**: roles → permissions; **dynamic** (role changes apply on the next request).
- 🧩 **Role management**: create/update/delete roles, grant/revoke permissions, assign/revoke roles.
- 🚪 **API Gateway**: single public entrypoint, REST→gRPC, per-route authorization.
- 📦 **Ready to run**: Docker Compose + Kubernetes manifests, auto migrations & seed, bootstrap admin.
- ✅ **Verified**: end-to-end smoke test + Postman/Bruno collections.

## Architecture

```
client ──REST──▶ Gateway (Gin) ──gRPC──▶ Auth Service ──▶ Postgres (auth_db)
                     │            └─gRPC──▶ User Service ──▶ Postgres (user_db)
                     │                          ▲
                     │   register / delete      │ consumes
                     └ validates JWT, RBAC      │
                                                │
        Auth ──outbox──▶ NATS JetStream ──iam.user.*──┘   (async, eventually consistent)
```

Auth and User never call each other: cross-service effects (profile create on
register, profile delete on delete) flow through a **transactional outbox →
NATS JetStream → idempotent consumer**. Full diagrams & flows:
**[docs/en/architecture.md](docs/en/architecture.md)**.

## Quick start

```bash
make up                 # pull service images from GHCR + run the full stack
make up IMAGE_TAG=dev   # or use locally-built images
make smoke              # end-to-end smoke test against http://localhost:8080
make down               # stop + remove volumes
```

**Observability** (started with the stack): distributed traces in **Jaeger**
([localhost:16686](http://localhost:16686)) — every request is traced gateway →
auth/user → Postgres (with SQL spans); metrics in **Prometheus**
([localhost:9090](http://localhost:9090)) and a **Grafana** "IAM Overview"
dashboard ([localhost:3000](http://localhost:3000)). Requests carry an
`X-Request-Id` correlated into logs and traces.

**Interactive API**: open **[localhost:8080/docs](http://localhost:8080/docs)**
for live **Swagger UI** — log in via `POST /auth/login`, click **Authorize**,
and try every endpoint. The OpenAPI spec is at `/openapi.yaml`.

A bootstrap admin (`admin@iam.local` / `admin12345`) is created on first boot.
Then:

```bash
# register, log in, and see your roles & permissions
curl -s localhost:8080/auth/register -H 'Content-Type: application/json' \
  -d '{"email":"alice@iam.local","password":"alicepass123"}'
TOKEN=$(curl -s localhost:8080/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"alice@iam.local","password":"alicepass123"}' | jq -r .access_token)
curl -s localhost:8080/me -H "Authorization: Bearer $TOKEN"
```

## Live demo & benchmark

Both stacks run live on **k3s via ArgoCD (GitOps)**, side by side, behind Cloudflare:

- **Go** — interactive Swagger: **https://iam-go.digitalglobalgrowth.com/docs/**
- **Rust** — interactive Swagger: **https://iam-rust.digitalglobalgrowth.com/docs/**

Log in with `admin@iam.local` / `ChangeMeAdmin-2026` (Authorize → Bearer), then
try any endpoint. The same k6 load runs against both for a Go-vs-Rust comparison
— see **[BENCHMARKS.md](BENCHMARKS.md)** (`bench/load.js`).

## API

REST on `:8080`. Highlights: `/auth/{register,login,refresh,logout}`, `/me`,
`/users[/:id]`, `/roles`, `/permissions`, role/permission management. Full
reference with examples & error codes: **[docs/en/api-reference.md](docs/en/api-reference.md)**.

Try it with Postman or Bruno — see **[docs/en/api-collections.md](docs/en/api-collections.md)**.

## Project structure

This umbrella repo holds only the platform layer; service code lives in the
[per-service repos](#repositories).

```
deploy/       docker-compose · k8s · .env.example
docs/         en/ · id/ (bilingual)
scripts/      smoke.sh
*.json        Postman collection + environment
```

## Documentation

Full bilingual docs in **[`docs/`](docs/en/README.md)**: Architecture · API
Reference · RBAC · Deployment · Development (with DB ERD) · API Collections.

## Development

Each service is developed in its own repo (`make build` / `make test` /
`make docker` there). For cross-repo work, check the repos out side by side and
span them with a `go.work` (kept out of git). The contracts and libs repos are
tagged; services pin exact versions. Details:
**[docs/en/development.md](docs/en/development.md)**.

## Deployment

Docker Compose (local) and Kubernetes (kustomize) — see
**[docs/en/deployment.md](docs/en/deployment.md)**.

## Roadmap

- [x] Rate limiting on auth endpoints (per-IP fixed window, 60/min)
- [ ] Audit log for RBAC changes
- [ ] OpenAPI/Swagger spec generation
- [ ] Refresh-token reuse detection

## Contributing

Contributions welcome! See **[CONTRIBUTING.md](CONTRIBUTING.md)** and our
**[Code of Conduct](CODE_OF_CONDUCT.md)**.

## License

[MIT](LICENSE) © 2026 malvin
