# Deployment & Operasi — iam-go

🌐 [English](../en/deployment.md) | **Bahasa Indonesia** · [↑ Indeks dokumentasi](README.md)

## Docker Compose (lokal)

```bash
cd deploy
cp .env.example .env
docker compose up --build -d
# ... use it ...
docker compose down -v        # stop + remove volumes
```

Layanan: `postgres`, `auth`, `user`, `gateway` (mengekspos `:8080`). Auth & user
masing-masing menjalankan migrasi + seed RBAC saat startup; auth juga membuat admin
bootstrap.

### Environment variable (`deploy/.env.example`)

| Variable | Default | Digunakan oleh |
|---|---|---|
| `POSTGRES_USER` / `POSTGRES_PASSWORD` | `app` / `app_secret` | postgres |
| `AUTH_DATABASE_URL` | `postgres://app:app_secret@postgres:5432/auth_db?sslmode=disable` | auth |
| `USER_DATABASE_URL` | `postgres://app:app_secret@postgres:5432/user_db?sslmode=disable` | user |
| `JWT_SECRET` | `change-me-...` | auth |
| `JWT_ISSUER` | `iam-auth` | auth |
| `ACCESS_TOKEN_TTL` | `900` (d) | auth |
| `REFRESH_TOKEN_TTL` | `604800` (d) | auth |
| `BOOTSTRAP_ADMIN_EMAIL` | `admin@iam.local` | auth |
| `BOOTSTRAP_ADMIN_PASSWORD` | `admin12345` | auth |
| `AUTH_GRPC_PORT` / `USER_GRPC_PORT` | `50051` / `50052` | auth / user |
| `AUTH_GRPC_ADDR` / `USER_GRPC_ADDR` | `auth:50051` / `user:50052` | gateway |
| `GATEWAY_HTTP_PORT` | `8080` | gateway |
| `LOG_LEVEL` | `info` | semua |

> **Produksi**: ubah `JWT_SECRET`, kredensial DB, dan kata sandi admin bootstrap.
> Letakkan secret yang sebenarnya di secret manager, bukan di `.env`.

## Kubernetes

Manifest berada di `deploy/k8s` (kustomize). Build & muat image terlebih dahulu
(mis. ke kind/minikube), lalu:

```bash
kubectl apply -k deploy/k8s
kubectl -n iam-go port-forward svc/gateway 8080:8080
../scripts/smoke.sh http://localhost:8080
```

Mencakup: `Namespace` (`iam-go`), `Secret` (JWT secret, kredensial DB, password
bootstrap, URL DB), `ConfigMap` (konfigurasi non-secret + SQL init postgres),
Postgres (Deployment + PVC + Service), ketiga layanan (Deployment + Service), dan
sebuah `Ingress` (host `iam-go.local`). Auth & user menggunakan probe readiness/liveness
**gRPC**; gateway menggunakan probe HTTP pada `/healthz`.

## Migrasi & seed

Migrasi disematkan (`go:embed`) dan diterapkan saat startup layanan oleh sebuah
runner kecil yang melacak versi yang telah diterapkan di `schema_migrations`. Tidak
ada CLI eksternal yang dibutuhkan. File berada di `services/{auth,user}/db/migrations`
(penamaan golang-migrate `*.up.sql` / `*.down.sql`). Seed RBAC dan permission
`role:write` juga merupakan migrasi.

## Health check

- Gateway: `GET /healthz` → `{"status":"ok"}`.
- Auth/User: gRPC health (`grpc_health_v1`), digunakan oleh probe K8s.

## Pemecahan masalah

| Gejala | Kemungkinan penyebab / solusi |
|---|---|
| Layanan keluar dengan "postgres not reachable" | Postgres masih booting; layanan mencoba ulang ~30d. Periksa `docker compose logs postgres`. |
| `401 missing bearer token` | Header `Authorization` tidak ada/tidak valid; login terlebih dahulu. |
| `403 permission denied` | Role token tidak memiliki permission yang dibutuhkan (wajar untuk non-admin). |
| Port 8080/5432 sudah terpakai | Stack lain (`iam-rust`) sedang berjalan — jalankan `make down` terlebih dahulu. |
| Perubahan pada proto/SQL tidak tercermin | Jalankan ulang `make proto` / `make sqlc`, lalu rebuild image. |
---

## Pengerasan keamanan

Variabel environment tambahan:

| Variabel | Default | Tujuan |
|---|---|---|
| `APP_ENV` | `development` | Set ke `production` untuk mengaktifkan guard keamanan |
| `INTERNAL_SERVICE_TOKEN` | (nilai dev) | Secret bersama yang dikirim gateway ke service internal; service menolak panggilan tanpa token ini |

Postgres hanya dipublish di `127.0.0.1` (localhost), tidak ke luar.

### Checklist pengerasan produksi

- Set **`APP_ENV=production`** — service menolak start bila `JWT_SECRET` masih
  default, password admin default, atau `INTERNAL_SERVICE_TOKEN` kosong.
- Set **`JWT_SECRET`** kuat (>= 32 byte), **`BOOTSTRAP_ADMIN_PASSWORD`** non-default,
  dan **`INTERNAL_SERVICE_TOKEN`** asli.
- Aktifkan **TLS** di semua lapis: ingress, gRPC TLS/mTLS gateway↔service, dan Postgres SSL.
- **`NetworkPolicy`** Kubernetes bawaan hanya mengizinkan gateway menjangkau
  service auth/user, dan hanya service yang menjangkau Postgres.
- Endpoint auth **di-rate-limit**; access token **bisa dicabut** via denylist jti (logout).
### TLS & secret (opsional, default mati)

- **TLS/mTLS** — buat sertifikat lokal dengan `./scripts/gen-certs.sh` (menulis
  ke `deploy/tls/`). TLS bersifat opt-in (`TLS_ENABLED`); tanpa itu semua jalan
  plaintext seperti sebelumnya. Untuk produksi pakai cert-manager (ingress) dan
  TLS/mTLS gRPC antara gateway dan service.
- **Manajemen secret** — `Secret` K8s bawaan hanya placeholder. Untuk produksi
  ganti dengan salah satu: **Sealed Secrets** (enkripsi secret ke git),
  **External Secrets Operator** (sinkron dari secret manager), atau **HashiCorp
  Vault**. Semua drop-in: nama env var tetap sama, hanya sumber nilainya berbeda.
