# Changelog

All notable changes to **iam-go** are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.10.0] - 2026-06-11

Multi-tenant + multi-project — the IAM becomes a B2B identity platform
(Organizations). Full doc: [docs/en/multi-tenant.md](docs/en/multi-tenant.md).

### Added
- **Tenants, projects, memberships** — one global user belongs to many tenants;
  per-tenant roles, members, OAuth clients, API keys. A seeded default tenant
  (`…0001`) backfills all existing data, so single-tenant stays unchanged.
- **Tenant-bound tokens** — access token carries `tenant_id`/`project_id`; login
  auto-picks the user's active membership and refresh preserves it.
- **`POST /auth/switch`** + **`GET /me/memberships`** — switch the active tenant
  (re-issues a token; 403 if not a member) and list the caller's tenants.
- **Tenant / project / member admin** — `POST|GET /tenants`, `POST|GET /projects`,
  `GET|POST /members` + `DELETE /members/:id`, gated by `tenant:*`/`project:*`/
  `member:*`. Creating a tenant makes the creator its admin.
- **Scoped RBAC** — roles/permissions resolved per the token's tenant + project;
  permission cache keyed `perms:{tenant}:{project}:{user}`.
- **Postgres Row-Level Security (fail-closed)** on the tenant tables via a
  non-superuser `iam_rls` role — DB-level isolation on top of app-layer `WHERE`.
- **OIDC client → tenant binding** — a login through an OAuth client is scoped to
  that client's tenant (members only); `RegisterClient` stamps the caller tenant.
- **Batch `GetProfiles`** (user service) — `/users` is now the active-tenant
  directory (members joined with profiles in one call, no N+1).
- **Console**: tenant switcher + Tenants/Projects/Members pages.

### Fixed
- **Refresh-token rotation grace** — concurrent refreshes with the same token
  (e.g. NextAuth firing several requests when the access token expires) wiped the
  whole token family via reuse-detection, locking the session out despite a valid
  7-day refresh token. A rotated token re-presented within a 60s grace is now
  re-issued; logout-revoked tokens still hard-fail.

### Ops
- No new env/secret. Migrations `000010`–`000013` run on auth startup (apply by
  rolling out auth, then user + gateway). Silenced per-probe access logs
  (gRPC health checks).

## [0.9.2] - 2026-06-11

### Added
- **2FA status** (`GET /auth/2fa` → GetTotpStatus) — reports whether 2FA is
  active; the console uses it to hide enrollment once 2FA is on.
- **Integration tests** against a real Postgres via testcontainers for the auth
  repo (Go `-tags=integration`, Rust `--features integration`), run in CI.
- **Saga for permanently-failed profile creation** — after N retries the user
  service emits `ProfileCreationFailed`; auth records it and the profile
  self-heals on the next `GET /users/me` (forward recovery — the user is never
  locked out of an account they registered).

### Fixed
- **2FA in the OIDC `/authorize` browser flow** — a 2FA-enabled user got
  "login failed"; the flow now prompts for a TOTP/recovery code
  (`/authorize/totp`) before issuing the authorization code.
- **Re-enrolling 2FA when already enabled** reset the secret and silently
  disabled 2FA; it is now rejected (409 — disable first).

### Changed
- **`ListRoles` N+1 removed** — roles + their permissions load in one
  `LEFT JOIN + array_agg` query.

## [0.9.1] - 2026-06-11

### Added
- **Bulk role assignment** — `POST /roles/{name}/assignments` assigns one role to
  many users in a single call (partial success: invalid ids come back in `failed[]`).
- **Soft-deleted user view** — `GET /users?deleted=true` lists soft-deleted
  profiles so they can be restored.
- **Redis token denylist + permission cache** (shared across auth replicas):
  logout denylists the access-token jti in Redis so other replicas reject it at
  once, and `ValidateToken` caches a user's permissions (short TTL, invalidated
  on role change) to cut the RBAC join off the hot path. Falls back to Postgres /
  no-cache when `REDIS_URL` is unset.
- **Full admin console** — Users gain soft-delete, a restore view (`?deleted`),
  and bulk role assignment; Roles gain create/delete + grant/revoke permission.

### Changed
- ROADMAP brought in sync: v0.7 (OIDC) sub-items marked shipped; v0.8 Redis
  denylist + permission cache and v0.9 bulk operations completed.

## [0.9.0] - 2026-06-11

### Added (v0.9 — M4: enterprise auth)
- **2FA / TOTP** — opt-in self-service: `POST /auth/2fa/enroll` (secret + otpauth
  URI + one-time recovery codes), `/activate`, `/disable`. Login becomes a
  challenge: a password login returns `mfa_required` + a short-lived `mfa_token`,
  then `POST /auth/login/totp` with a TOTP or recovery code issues the token pair.
- **Scoped API keys** (`iamk_…`) — `POST`/`GET`/`DELETE /api-keys`. Requested
  scopes must be a subset of the creator's permissions; the effective scope is
  `requested ∩ the owner's current permissions`. Stored as a SHA-256 hash, with
  optional expiry and revoke. The gateway authenticates `Authorization: Bearer
  iamk_…` via ValidateApiKey.
- **Soft-delete + restore** — `deleted_at` on identity and profile.
  `DELETE /users/:id` soft-deletes (login blocked, profile hidden, sessions
  revoked); `POST /users/:id/restore` reverses it; `DELETE /users/:id?hard=true`
  removes permanently. Driven by `UserDeleted{hard}` / `UserRestored` events.

### Changed
- Access-token validation now rejects soft-deleted accounts and MFA-purpose tokens.

## [0.8.0] - 2026-06-11

### Added (v0.8 — horizontal scale)
- **Redis-backed rate limiter** — the per-IP auth limiter now counts in Redis
  (atomic `INCR`+`EXPIRE`), so the cap is enforced **globally across gateway
  replicas** instead of per-pod. Falls back to in-memory when `REDIS_URL` is
  unset (single-instance/dev). Fail-open on Redis error.
- **Multi-replica gateway** — the demo cluster runs the gateway at **2 replicas**
  behind a shared, ephemeral **Redis**; both stacks gain a `redis` Deployment and
  `REDIS_URL` config.

### Changed
- `BENCHMARKS.md` gains a **Horizontal scale** section proving the shared cap
  (5 global vs 5×2 per-pod: 8 requests → 5 passed, 3× `429`).

## [0.7.0] - 2026-06-10

### Added (v0.7 — OIDC / OAuth2 provider)
- **OpenID Connect provider** at the gateway: discovery
  (`/.well-known/openid-configuration`), **JWKS** (`/.well-known/jwks.json`),
  **Authorization Code + PKCE** flow (`/authorize` with a login form + consent
  screen + signed session cookie), **token endpoint** (`/token` —
  `authorization_code` & `refresh_token`, `client_secret_basic`/`_post`/PKCE),
  **UserInfo** (`/userinfo`), and **client registration** (`/oauth/clients`).
- **ID tokens** (RS256) with `iss`/`sub`/`aud`/`email`/`nonce`, verifiable by
  relying parties via JWKS.
- A demo confidential **console client** is seeded on first boot
  (`OIDC_CONSOLE_*` env).
- Swagger UI documents all OIDC endpoints.

### Changed
- **Token signing migrated HS256 → RS256** (asymmetric, rotatable keys with
  `kid`); token validation stays centralized in the auth service.

### Notes
- New deploy env: `OIDC_ISSUER`, `SESSION_SECRET`, `OIDC_CONSOLE_*`.

## [0.6.5] - 2026-06-10

### Changed
- **ROADMAP refreshed** to match reality: v0.5 (Observability) and v0.6 (Show it
  off — Swagger, live k3s demo, k6 benchmark, configurable rate limit) marked
  shipped; next milestones planned — **v0.7 OIDC/OAuth2 provider**, **v0.8
  horizontal scale + Redis**.

## [0.6.4] - 2026-06-10

### Changed
- **BENCHMARKS.md: added rate-limiter-OFF results.** With `AUTH_RATE_LIMIT=0`
  the mixed load runs at **0% errors** (final proof the earlier ~9% was the auth
  rate limiter, not argon2), and the Go/Rust ranking flips: **Go** leads the
  I/O-bound read path, **Rust** leads the CPU-bound argon2 login path.

## [0.6.3] - 2026-06-10

### Added
- **Configurable auth rate limit**: the gateway's per-IP `/auth/*` limiter now
  reads `AUTH_RATE_LIMIT` (requests, default 60) and `AUTH_RATE_WINDOW_SECONDS`
  (window seconds, default 60); `AUTH_RATE_LIMIT=0` disables it. Was hard-coded.

## [0.6.2] - 2026-06-10

### Fixed
- **BENCHMARKS.md analysis corrected**: the ~9% error under mixed load is the
  gateway's per-IP **auth rate limiter** (HTTP 429, 60 req/min on `/auth/*`) —
  brute-force protection working as designed — **not** argon2 saturation (the
  v0.6.1 note was wrong). Added a login-only status-code breakdown (60 logins
  allowed, then 429) confirming it.

### Changed
- Roadmap: marked **rate limiting on auth endpoints** done — it already ships.

## [0.6.1] - 2026-06-10

### Changed
- **Live demo is up**: README points at the running stacks on k3s via ArgoCD —
  `https://iam-go.digitalglobalgrowth.com/docs/` and
  `https://iam-rust.digitalglobalgrowth.com/docs/` (interactive Swagger UI).
- **BENCHMARKS.md filled with real numbers**: Go-vs-Rust head-to-head on the
  same single-node cluster (off-node k6, no Cloudflare). Read-path 649 vs 631
  req/s, p95 17.5 vs 21.6 ms, 0% errors — within ~3%; argon2 + infra sizing
  dominate, not the language. Added an in-cluster k6 Job (`iam-gitops/bench`).

## [0.6.0] - 2026-06-10

### Added (v0.6 — Show it off)
- **Interactive API**: the gateway serves **Swagger UI** at `/docs` and the
  OpenAPI 3 spec at `/openapi.yaml` (vendored, no CDN). Authorize with a Bearer
  token and try every endpoint live — no separate frontend needed.
- **Live demo via GitOps**: deployable to **k3s with ArgoCD** — see the
  `iam-gitops` repo (ArgoCD Applications + kustomize overlays + Traefik ingress),
  both stacks side by side.
- **Go-vs-Rust benchmark harness**: `bench/load.js` (k6) + `BENCHMARKS.md`
  (methodology + comparison table) for a fair head-to-head on identical infra.

## [0.5.0] - 2026-06-09

### Added (v0.5 — Observability)
- **Distributed tracing** (OpenTelemetry → Jaeger): every request is traced
  across gateway → auth/user → Postgres, including SQL spans (otelpgx). Optional
  via `OTEL_EXPORTER_OTLP_ENDPOINT`.
- **Prometheus metrics**: gateway exposes `/metrics` (HTTP histograms); auth/user
  expose gRPC metrics on `:9100`. Scraped by a bundled Prometheus.
- **Grafana** with a provisioned Prometheus datasource and an "IAM Overview"
  dashboard (request rate, p95 latency, gRPC calls, error rate).
- **Correlation IDs**: the gateway assigns/propagates `X-Request-Id` end-to-end
  (HTTP → gRPC metadata) and includes `request_id` + `trace_id` in structured logs.
- Compose adds `jaeger` (UI :16686), `prometheus` (:9090), `grafana` (:3000).

## [0.4.0] - 2026-06-09

### Changed (v0.4 — True microservices)
- **Separate repositories per service**: this repo is now the platform/umbrella;
  each service lives in its own repo
  ([iam-go-gateway](https://github.com/malvinpratama/iam-go-gateway),
  [iam-go-auth](https://github.com/malvinpratama/iam-go-auth),
  [iam-go-user](https://github.com/malvinpratama/iam-go-user)) with shared
  module repos ([iam-go-contracts](https://github.com/malvinpratama/iam-go-contracts),
  [iam-go-libs](https://github.com/malvinpratama/iam-go-libs)). Each is built,
  versioned and deployed independently.
- **One database instance per service** (`postgres-auth`, `postgres-user`)
  instead of a single shared instance.
- **Event-driven cross-service flow**: register/delete no longer orchestrate
  synchronously at the gateway. Auth writes a **transactional outbox** in the
  same DB transaction; a relay publishes to **NATS JetStream**; the user service
  consumes idempotently to create/drop the profile. `GET /users/me` lazy-heals as
  the eventual-consistency safety net. The broker is optional (`NATS_URL`).
- **CI/CD per repo** (GitHub Actions): lint + test, and service images published
  to GHCR; `buf lint`/`breaking` on contracts. Umbrella compose pulls the images.

### Future work
- Compensation saga (`iam.user.registration_failed`) for permanently-failed
  profile creation — not needed today thanks to idempotent upsert + lazy heal.

### Added (v0.2 — Security+)
- **Account recovery**: email verification (`/auth/verify-email/request`,
  `/auth/verify-email`) and password reset (`/auth/password-reset/request`,
  `/auth/password-reset`). In non-production the token is returned as `dev_token`;
  otherwise it is emailed (pluggable sender; default logs the message).
- **Audit log** of sensitive actions (`audit_events`), readable at `GET /audit`
  (`audit:read`, admin).
- **Account lockout**: lock after `LOGIN_MAX_FAILURES` failed logins for
  `LOGIN_LOCKOUT_SECONDS` (configurable; `0` disables).
- **Refresh-token reuse detection**: presenting an already-revoked refresh token
  revokes the user's whole token family.
- Optional **email-verification gate** on login (`REQUIRE_EMAIL_VERIFICATION`).
- Opt-in **TLS** cert generator (`scripts/gen-certs.sh`) + production hardening
  and secrets-management docs (Vault / Sealed Secrets / External Secrets).
- All toggles default to non-breaking; the existing smoke flow is unchanged.

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
