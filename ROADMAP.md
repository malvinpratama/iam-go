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

## 🟢 v0.2 — Security+ (mostly shipped)

- 🟢 **Refresh-token reuse detection** → revokes the whole session family. **S**
- 🟢 **Audit log** for sensitive actions, readable at `GET /audit`. **M**
- 🟢 **Per-account lockout** after N failed logins (configurable). **S**
- 🟢 **Email verification** + **password reset** (single-use tokens; dev returns the token, prod emails it). **M**
- 🟡 **TLS/mTLS** — opt-in cert generator (`scripts/gen-certs.sh`) + docs; wire per deployment. **M**
- 🟡 **Secrets via Vault / Sealed Secrets** — documented opt-in (default uses K8s Secret). **M**

## 🔜 v0.3 — Quality & observability

- 🟢 **CI** (GitHub Actions): build, unit tests, and per-service `docker build`
  publishing images to GHCR (shipped per-repo in v0.4). **M**
- 🟢 **OpenTelemetry** distributed tracing (gateway → auth/user → DB, incl. SQL spans). **M**
- 🟢 **Prometheus metrics** (`/metrics`) + Grafana dashboard (latency, login
  failures, RPS). **M**
- 🔜 **Integration tests** with testcontainers (Postgres) for repos/handlers. **M**
- 🔜 **Correlation/request IDs** propagated through gRPC metadata + logs. **S**
- 🔜 Fix `ListRoles` N+1 query. **S**

## 🟢 v0.4 — True microservices (shipped)

- 🟢 **Separate repository per service** (gateway/auth/user) + shared contracts
  and libs repos; each built, versioned (semver tags) and deployed independently. **L**
- 🟢 **One database instance per service** (`postgres-auth`, `postgres-user`). **S**
- 🟢 **Event-driven** register/delete via a **transactional outbox → NATS
  JetStream → idempotent consumer** (no synchronous cross-service writes). **L**
- 🟢 **CI/CD per repo** (GitHub Actions): build/test + service images published
  to GHCR; the umbrella compose pulls them. **M**
- 🔮 Compensation saga for permanently-failed profile creation (future). **S**

## 🔮 v0.5 — Features

- 🔮 **OIDC / OAuth2 provider** (discovery, authorization code + PKCE; social
  login). The flagship "wow" feature. **L**
- 🔮 **2FA / TOTP** (and recovery codes), required for admins. **M**
- 🔮 **API keys / service accounts** for non-human auth. **M**
- 🔮 **Soft-delete + restore**, and enforce the `status` field (suspend blocks login). **S**
- 🔮 Bulk operations (assign role to many users). **S**

## 🔮 v0.6 — Scale & polish

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

## 🟢 v0.2 — Keamanan lanjutan (sebagian besar rilis)

- 🟢 **Deteksi reuse refresh-token** → cabut seluruh sesi terkait. **S**
- 🟢 **Audit log** aksi sensitif, dibaca via `GET /audit`. **M**
- 🟢 **Lockout per-akun** setelah N login gagal (bisa dikonfigurasi). **S**
- 🟢 **Verifikasi email** + **reset password** (token sekali pakai; dev mengembalikan token, prod via email). **M**
- 🟡 **TLS/mTLS** — generator sertifikat opt-in (`scripts/gen-certs.sh`) + docs; aktifkan per deployment. **M**
- 🟡 **Secret via Vault / Sealed Secrets** — opt-in terdokumentasi (default pakai K8s Secret). **M**

## 🔜 v0.3 — Kualitas & observability

- 🟢 **CI** (GitHub Actions): build, unit test, dan `docker build` per service
  yang mem-publish image ke GHCR (rilis per-repo di v0.4). **M**
- 🟢 **OpenTelemetry** tracing terdistribusi (gateway → auth/user → DB). **M**
- 🟢 **Prometheus metrics** (`/metrics`) + dashboard Grafana. **M**
- 🔜 **Integration test** dengan testcontainers (Postgres). **M**
- 🔜 **Correlation/request ID** diteruskan lewat metadata gRPC + log. **S**
- 🔜 Perbaiki N+1 di `ListRoles`. **S**

## 🟢 v0.4 — True microservices (sudah rilis)

- 🟢 **Repo terpisah per service** (gateway/auth/user) + repo contracts & libs
  bersama; tiap repo di-build, di-versioning (tag semver), dan di-deploy independen. **L**
- 🟢 **Satu instance database per service** (`postgres-auth`, `postgres-user`). **S**
- 🟢 **Event-driven** register/delete lewat **outbox transaksional → NATS
  JetStream → konsumen idempoten** (tanpa panggilan sinkron antar-service). **L**
- 🟢 **CI/CD per repo** (GitHub Actions): build/test + image service ke GHCR;
  compose umbrella menariknya. **M**
- 🔮 Saga kompensasi untuk pembuatan profil yang gagal permanen (ke depan). **S**

## 🔮 v0.5 — Fitur

- 🔮 **OIDC / OAuth2 provider** (discovery, authorization code + PKCE; social
  login). Fitur unggulan. **L**
- 🔮 **2FA / TOTP** (+ recovery code), wajib untuk admin. **M**
- 🔮 **API key / service account** untuk auth non-manusia. **M**
- 🔮 **Soft-delete + restore**, dan menegakkan field `status` (suspend memblok login). **S**
- 🔮 Operasi massal (assign role ke banyak user). **S**

## 🔮 v0.6 — Skala & poles

- 🔮 **Redis** untuk denylist token, rate-limit, dan cache permission
  (siap multi-instance). **M**
- 🔮 **OpenAPI/Swagger** spec + Swagger UI di gateway. **M**
- 🔮 **Load test** (k6) dengan angka + perbandingan Go vs Rust. **S**
- 🔮 **Helm chart** (alternatif kustomize); Makefile root untuk kedua stack. **S**
- 🔮 **Live demo** (Fly.io/Railway) ditautkan dari README. **S**
