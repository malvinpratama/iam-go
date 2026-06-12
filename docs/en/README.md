# iam-go Documentation

🌐 **English** | [Bahasa Indonesia](../id/README.md) · [← Project README](../../README.md)

Identity & Access Management — **Auth + User microservices with granular RBAC**,
built in **Go** (Gin · gRPC · sqlc · PostgreSQL · JWT).

## Contents

| Doc | What's inside |
|---|---|
| [Architecture](architecture.md) | Services, component & sequence diagrams, token model |
| [API Reference](api-reference.md) | Every REST endpoint (request/response/errors) + gRPC contracts |
| [RBAC Model](rbac.md) | Roles, permissions, seed, dynamic RBAC, role management |
| [Multi-tenancy](multi-tenant.md) | Tenants/projects/memberships, tenant-bound tokens, scoped RBAC, RLS, OIDC client→tenant |
| [Security](security.md) | Auth/sessions, 2FA-at-rest, OIDC, tenant isolation, secrets, network & container hardening |
| [Deployment & Ops](deployment.md) | Docker Compose, Kubernetes, env vars, migrations, troubleshooting |
| [Development](development.md) | Toolchain, codegen, project structure, tests, **DB ERD** |
| [API Collections](api-collections.md) | Postman & Bruno usage (two native collections) |

## Quick start

```bash
make up        # build + run the full stack (postgres + auth + user + gateway)
make smoke     # end-to-end smoke test against http://localhost:8080
make down      # stop + remove volumes
```

A bootstrap admin (`admin@iam.local` / `admin12345`) is created on first boot.
Register a user, log in, and call `GET /me` to see your roles & permissions.

## At a glance

```
client ──REST──▶ Gateway (Gin) ──gRPC──▶ Auth Service ──▶ Postgres (auth_db)
                     │            └─gRPC──▶ User Service ──▶ Postgres (user_db)
                     └ validates JWT, resolves permissions, enforces RBAC per route
```

The parallel Rust implementation lives in iam-rust (https://github.com/malvinpratama/iam-rust).
