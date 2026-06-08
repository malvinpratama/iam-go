# Changelog

All notable changes to **iam-go** are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Auth service (gRPC): register, login, refresh (with rotation), logout
  (revocation), `ValidateToken`.
- User service (gRPC): profile create/get/update/delete, paginated list/search.
- API Gateway (Gin, REST→gRPC) with JWT auth middleware and per-route RBAC.
- Granular RBAC: roles + permissions, seeded `admin`/`user`.
- `GET /me` — caller's own roles & permissions.
- `GET /permissions` — list all permissions (`role:read`).
- Role management: `POST/PATCH/DELETE /roles`, grant/revoke permission to a role
  (`role:write`); built-in roles protected from deletion.
- Assign/revoke role to a user (`role:assign`).
- JWT access + refresh tokens; refresh tokens hashed & revocable in the DB.
- Embedded migrations + RBAC seed run at startup; bootstrap admin on first boot.
- Docker Compose + Kubernetes (kustomize) manifests; health checks.
- End-to-end smoke test (`scripts/smoke.sh`).
- Postman & Bruno API collections.
- Bilingual documentation (English + Indonesian) under `docs/`.

### Security
- Fixed broken object-level authorization: editing another user's profile now
  requires `user:write` (admin); `profile:write` covers only your own profile.
- `DELETE /users/:id` now deletes the identity (credentials, roles, refresh
  tokens) as well as the profile — a deleted user can no longer log in.
- Defense in depth: internal services re-check permissions and require a shared
  `INTERNAL_SERVICE_TOKEN` from the gateway; Kubernetes `NetworkPolicy` restricts
  service-to-service traffic.
- Access tokens are now revocable: logout denylists the token by `jti`.
- Constant-time login (dummy hash on unknown users) to reduce user enumeration.
- Per-IP rate limiting on `/auth/*`; request body-size limit.
- Startup security guard (rejects default JWT secret / admin password / missing
  internal token when `APP_ENV=production`); gin release mode + gRPC reflection
  disabled in production; Postgres bound to localhost.
- gRPC panic-recovery interceptor; passwords hashed with **Argon2id** (parity with Rust).

[Unreleased]: https://github.com/malvin/iam-go
