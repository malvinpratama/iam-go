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

- 🔐 **Auth**: register, login, JWT access + refresh tokens with **rotation** (reuse-detection with a grace window) and **revocation**; self-service **password change**; account lockout on brute force.
- 🔑 **2FA / TOTP**: opt-in authenticator-app 2FA with one-time recovery codes; the shared secret is **encrypted at rest** (AES-256-GCM).
- 🪪 **OIDC / OAuth2 provider**: Authorization Code + **PKCE**, discovery document, JWKS, `/userinfo`, dynamic client registration, RP-initiated logout — log into the console (or any RP) via the IAM's own flow.
- 🎫 **Scoped API keys**: `iamk_…` keys (SHA-256 hashed), scopes ⊆ the owner's current permissions.
- 🛡️ **Granular RBAC**: roles → permissions; **dynamic** (role changes apply on the next request); **scoped per tenant/project**; full role/permission management.
- 🏢 **Multi-tenant** (v0.10): tenants/projects/memberships, tenant-bound tokens + switcher, OIDC client→tenant, app-layer **+ Postgres RLS** isolation — see **[docs/en/multi-tenant.md](docs/en/multi-tenant.md)**.
- 🔒 **Security-hardened** (v0.11): encrypted 2FA secrets, fail-closed internal auth, RLS-enforced writes, gateway edge hardening, Sealed Secrets, default-deny NetworkPolicies, non-root + read-only containers, immutable image pins — see **[docs/en/security.md](docs/en/security.md)**.
- 👤 **Users**: profile CRUD + paginated search, via a dedicated service; audit log.
- 🚪 **API Gateway**: single public entrypoint, REST→gRPC, per-route authorization.
- 📦 **Ready to run**: Docker Compose + Kubernetes manifests, auto migrations & seed, bootstrap admin.
- ✅ **Verified**: end-to-end smoke test + **Postgres integration tests** (Testcontainers) + Postman/Bruno collections.

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

Log in with the **read-only demo account** `demo@iam.local` / `demo1234`
(Authorize → Bearer), then try any endpoint — it can read everything but cannot
modify anything. The admin console runs on top of both backends at
**https://console.digitalglobalgrowth.com** (same demo credentials), with a live
switch between the Go and Rust backend. The same k6 load runs against both for a
Go-vs-Rust comparison — see **[BENCHMARKS.md](BENCHMARKS.md)** (`bench/load.js`).

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
Reference · RBAC · **[Security](docs/en/security.md)** · Multi-tenant · Deployment ·
Development (with DB ERD) · API Collections.

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

- [x] Rate limiting on auth endpoints (Redis-backed, shared across replicas)
- [x] OIDC / OAuth2 provider — Authorization Code + PKCE (v0.7)
- [x] 2FA / TOTP, scoped API keys, soft-delete (v0.9)
- [x] Multi-tenant + multi-project with Postgres RLS (v0.10)
- [x] Audit log + refresh-token reuse detection + OpenAPI/Swagger UI
- [x] Security hardening — encrypted 2FA, Sealed Secrets, NetworkPolicies, image pinning (v0.11)
- [ ] mTLS between the gateway and services
- [ ] Egress NetworkPolicy + least-privilege DB connection-role cutover

## Contributing

Contributions welcome! See **[CONTRIBUTING.md](CONTRIBUTING.md)** and our
**[Code of Conduct](CODE_OF_CONDUCT.md)**.

## License

[MIT](LICENSE) © 2026 malvin
