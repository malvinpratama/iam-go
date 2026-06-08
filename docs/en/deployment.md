# Deployment & Operations — iam-go

🌐 **English** | [Bahasa Indonesia](../id/deployment.md) · [↑ Docs index](README.md)

## Docker Compose (local)

```bash
cd deploy
cp .env.example .env
docker compose up --build -d
# ... use it ...
docker compose down -v        # stop + remove volumes
```

Services: `postgres`, `auth`, `user`, `gateway` (exposes `:8080`). Auth & user
each run migrations + RBAC seed on startup; auth also creates the bootstrap admin.

### Environment variables (`deploy/.env.example`)

| Variable | Default | Used by |
|---|---|---|
| `POSTGRES_USER` / `POSTGRES_PASSWORD` | `app` / `app_secret` | postgres |
| `AUTH_DATABASE_URL` | `postgres://app:app_secret@postgres:5432/auth_db?sslmode=disable` | auth |
| `USER_DATABASE_URL` | `postgres://app:app_secret@postgres:5432/user_db?sslmode=disable` | user |
| `JWT_SECRET` | `change-me-...` | auth |
| `JWT_ISSUER` | `iam-auth` | auth |
| `ACCESS_TOKEN_TTL` | `900` (s) | auth |
| `REFRESH_TOKEN_TTL` | `604800` (s) | auth |
| `BOOTSTRAP_ADMIN_EMAIL` | `admin@iam.local` | auth |
| `BOOTSTRAP_ADMIN_PASSWORD` | `admin12345` | auth |
| `AUTH_GRPC_PORT` / `USER_GRPC_PORT` | `50051` / `50052` | auth / user |
| `AUTH_GRPC_ADDR` / `USER_GRPC_ADDR` | `auth:50051` / `user:50052` | gateway |
| `GATEWAY_HTTP_PORT` | `8080` | gateway |
| `LOG_LEVEL` | `info` | all |

> **Production**: change `JWT_SECRET`, the DB credentials, and the bootstrap
> admin password. Put real secrets in a secret manager, not `.env`.

## Kubernetes

Manifests under `deploy/k8s` (kustomize). Build & load the images first
(e.g. into kind/minikube), then:

```bash
kubectl apply -k deploy/k8s
kubectl -n iam-go port-forward svc/gateway 8080:8080
../scripts/smoke.sh http://localhost:8080
```

Includes: `Namespace` (`iam-go`), `Secret` (JWT secret, DB creds, bootstrap pw,
DB URLs), `ConfigMap` (non-secret config + postgres init SQL), Postgres
(Deployment + PVC + Service), the three services (Deployment + Service), and an
`Ingress` (host `iam-go.local`). Auth & user use **gRPC** readiness/liveness
probes; gateway uses an HTTP probe on `/healthz`.

## Migrations & seed

Migrations are embedded (`go:embed`) and applied at service startup by a tiny
runner that tracks applied versions in `schema_migrations`. No external CLI is
needed. Files live in `services/{auth,user}/db/migrations` (golang-migrate
`*.up.sql` / `*.down.sql` naming). The RBAC seed and the `role:write` permission
are migrations too.

## Health checks

- Gateway: `GET /healthz` → `{"status":"ok"}`.
- Auth/User: gRPC health (`grpc_health_v1`), used by K8s probes.

## Troubleshooting

| Symptom | Likely cause / fix |
|---|---|
| Service exits with "postgres not reachable" | Postgres still booting; services retry ~30s. Check `docker compose logs postgres`. |
| `401 missing bearer token` | No/invalid `Authorization` header; log in first. |
| `403 permission denied` | The token's role lacks the required permission (expected for non-admins). |
| Port 8080/5432 already in use | The other stack (`iam-rust`) is running — `make down` it first. |
| Changes to proto/SQL not reflected | Re-run `make proto` / `make sqlc`, then rebuild images. |
---

## Security hardening

Extra environment variables:

| Variable | Default | Purpose |
|---|---|---|
| `APP_ENV` | `development` | Set to `production` to enforce security guards |
| `INTERNAL_SERVICE_TOKEN` | (dev value) | Shared secret the gateway presents to the internal services; services reject calls without it |

Postgres is published only on `127.0.0.1` (localhost), not externally.

### Production hardening checklist

- Set **`APP_ENV=production`** — services refuse to start with the default
  `JWT_SECRET`, the default admin password, or a missing `INTERNAL_SERVICE_TOKEN`.
- Set a strong **`JWT_SECRET`** (>= 32 bytes), a non-default
  **`BOOTSTRAP_ADMIN_PASSWORD`**, and a real **`INTERNAL_SERVICE_TOKEN`**.
- Enable **TLS** everywhere: ingress TLS, gRPC TLS/mTLS between the gateway and
  services, and Postgres SSL.
- The bundled Kubernetes **`NetworkPolicy`** lets only the gateway reach the
  auth/user services, and only the services reach Postgres.
- Auth endpoints are **rate-limited**; access tokens are **revocable** via the
  jti denylist (logout).
- The gateway runs in **gin release mode** and gRPC **reflection is disabled** when `APP_ENV=production`.
### TLS & secrets (optional, off by default)

- **TLS/mTLS** — generate local certs with `./scripts/gen-certs.sh` (writes
  `deploy/tls/`). TLS is opt-in (`TLS_ENABLED`); without it everything runs
  plaintext as before. For real deployments use cert-manager (ingress) and
  gRPC TLS/mTLS between gateway and services.
- **Secrets management** — the bundled K8s `Secret` is a placeholder. For
  production, swap it for one of: **Sealed Secrets** (encrypt secrets into git),
  **External Secrets Operator** (sync from a manager), or **HashiCorp Vault**
  (Agent Injector / CSI). All are drop-in: keep the same env var names, change
  only where the values come from. Nothing else needs to change.
