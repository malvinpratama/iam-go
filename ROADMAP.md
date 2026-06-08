# Roadmap — IAM (Go & Rust)

🌐 **English** | [Bahasa Indonesia](#roadmap-bahasa-indonesia)

This roadmap applies to **both** stacks (`iam-go`, `iam-rust`) — changes land in
both to keep parity. Legend: 🟢 done · 🔜 planned · 🔮 future · effort **S/M/L**.

## ✅ v0.1 — Foundation & hardening (shipped)

- 🟢 Auth service: register, login, JWT access + refresh (rotation), logout.
- 🟢 User service: profile CRUD + paginated search.
- 🟢 API gateway (REST→gRPC) with per-route authorization.
- 🟢 Granular RBAC: roles → permissions, dynamic; full role management.
- 🟢 Argon2id password hashing (both stacks), PostgreSQL, Docker Compose + K8s.
- 🟢 Bilingual docs (EN/ID), Postman + Bruno collections, smoke tests (16/16).
- 🟢 Security hardening: BOLA fix, real user deletion, defense-in-depth
  (internal service token + service-side permission re-check + NetworkPolicy),
  access-token revocation (jti denylist), constant-time login, per-IP rate
  limiting, body-size limit, startup secret guards.

## 🔜 v0.2 — Security+

- 🔜 **TLS/mTLS** everywhere (ingress, gateway↔service gRPC, Postgres SSL). **M**
- 🔜 **Refresh-token reuse detection** → revoke the whole session family. **S**
- 🔜 **Audit log** for sensitive actions (role/permission/user changes). **M**
- 🔜 **Per-account lockout** + exponential backoff (complements IP rate limit). **S**
- 🔜 **Email verification** + **password reset** (single-use tokens). **M**
- 🔜 Secrets via Vault / Sealed Secrets in K8s. **M**

## 🔜 v0.3 — Quality & observability

- 🔜 **CI** (GitHub Actions): build, unit tests, `golangci-lint`/`clippy`,
  `docker build`, run smoke tests for both stacks on every PR. **M**
- 🔜 **OpenTelemetry** distributed tracing (gateway → auth/user → DB). **M**
- 🔜 **Prometheus metrics** (`/metrics`) + Grafana dashboard (latency, login
  failures, RPS). **M**
- 🔜 **Integration tests** with testcontainers (Postgres) for repos/handlers. **M**
- 🔜 **Correlation/request IDs** propagated through gRPC metadata + logs. **S**
- 🔜 Fix `ListRoles` N+1 query. **S**

## 🔮 v0.4 — Features

- 🔮 **OIDC / OAuth2 provider** (discovery, authorization code + PKCE; social
  login). The flagship "wow" feature. **L**
- 🔮 **2FA / TOTP** (and recovery codes), required for admins. **M**
- 🔮 **API keys / service accounts** for non-human auth. **M**
- 🔮 **Soft-delete + restore**, and enforce the `status` field (suspend blocks login). **S**
- 🔮 Bulk operations (assign role to many users). **S**

## 🔮 v0.5 — Scale & polish

- 🔮 **Redis** for the token denylist, rate limiting, and permission cache
  (multi-instance ready). **M**
- 🔮 **OpenAPI/Swagger** spec + Swagger UI at the gateway. **M**
- 🔮 **Load tests** (k6) with published numbers + a Go-vs-Rust comparison. **S**
- 🔮 **Helm chart** (alternative to kustomize); root Makefile orchestrating both
  stacks. **S**
- 🔮 **Live demo** deploy (Fly.io/Railway) linked from the README. **S**

---

# Roadmap (Bahasa Indonesia)

🌐 [English](#roadmap--iam-go--rust) | **Bahasa Indonesia**

Roadmap ini berlaku untuk **kedua** stack (`iam-go`, `iam-rust`) — perubahan
diterapkan di keduanya agar tetap setara. Keterangan: 🟢 selesai · 🔜 direncanakan
· 🔮 ke depan · effort **S/M/L**.

## ✅ v0.1 — Fondasi & pengerasan (sudah rilis)

- 🟢 Auth service: register, login, JWT access + refresh (rotasi), logout.
- 🟢 User service: CRUD profil + pencarian berpaginasi.
- 🟢 API gateway (REST→gRPC) dengan otorisasi per-route.
- 🟢 RBAC granular: role → permission, dinamis; manajemen role lengkap.
- 🟢 Hash password Argon2id (kedua stack), PostgreSQL, Docker Compose + K8s.
- 🟢 Dokumentasi dwibahasa (EN/ID), koleksi Postman + Bruno, smoke test (16/16).
- 🟢 Pengerasan keamanan: fix BOLA, hapus user beneran, pertahanan berlapis
  (token internal antar-service + cek ulang permission di service + NetworkPolicy),
  pencabutan access token (denylist jti), login constant-time, rate-limit per IP,
  batas ukuran body, guard secret saat startup.

## 🔜 v0.2 — Keamanan lanjutan

- 🔜 **TLS/mTLS** di semua lapis (ingress, gRPC gateway↔service, Postgres SSL). **M**
- 🔜 **Deteksi reuse refresh-token** → cabut seluruh sesi terkait. **S**
- 🔜 **Audit log** untuk aksi sensitif (perubahan role/permission/user). **M**
- 🔜 **Lockout per-akun** + backoff eksponensial. **S**
- 🔜 **Verifikasi email** + **reset password** (token sekali pakai). **M**
- 🔜 Secret via Vault / Sealed Secrets di K8s. **M**

## 🔜 v0.3 — Kualitas & observability

- 🔜 **CI** (GitHub Actions): build, unit test, `golangci-lint`/`clippy`,
  `docker build`, jalankan smoke test kedua stack tiap PR. **M**
- 🔜 **OpenTelemetry** tracing terdistribusi (gateway → auth/user → DB). **M**
- 🔜 **Prometheus metrics** (`/metrics`) + dashboard Grafana. **M**
- 🔜 **Integration test** dengan testcontainers (Postgres). **M**
- 🔜 **Correlation/request ID** diteruskan lewat metadata gRPC + log. **S**
- 🔜 Perbaiki N+1 di `ListRoles`. **S**

## 🔮 v0.4 — Fitur

- 🔮 **OIDC / OAuth2 provider** (discovery, authorization code + PKCE; social
  login). Fitur unggulan. **L**
- 🔮 **2FA / TOTP** (+ recovery code), wajib untuk admin. **M**
- 🔮 **API key / service account** untuk auth non-manusia. **M**
- 🔮 **Soft-delete + restore**, dan menegakkan field `status` (suspend memblok login). **S**
- 🔮 Operasi massal (assign role ke banyak user). **S**

## 🔮 v0.5 — Skala & poles

- 🔮 **Redis** untuk denylist token, rate-limit, dan cache permission
  (siap multi-instance). **M**
- 🔮 **OpenAPI/Swagger** spec + Swagger UI di gateway. **M**
- 🔮 **Load test** (k6) dengan angka + perbandingan Go vs Rust. **S**
- 🔮 **Helm chart** (alternatif kustomize); Makefile root untuk kedua stack. **S**
- 🔮 **Live demo** (Fly.io/Railway) ditautkan dari README. **S**
