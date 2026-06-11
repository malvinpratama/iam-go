# Roadmap — IAM (Go & Rust)

🌐 **English** | [Bahasa Indonesia](#roadmap-bahasa-indonesia)

This roadmap applies to **both** stacks (`iam-go`, `iam-rust`) — changes land in
both to keep parity. Legend: 🟢 done · 🟡 partial/opt-in · 🔜 planned · 🔮 future
· effort **S/M/L**.

## ✅ v0.1 — Foundation & hardening (shipped)

- 🟢 Auth service: register, login, JWT access + refresh (rotation), logout.
- 🟢 User service: profile CRUD + paginated search.
- 🟢 API gateway (REST→gRPC) with per-route authorization.
- 🟢 Granular RBAC: roles → permissions, dynamic; full role management.
- 🟢 Argon2id password hashing (both stacks), PostgreSQL, Docker Compose + K8s.
- 🟢 Bilingual docs (EN/ID), Postman + Bruno collections, smoke tests.
- 🟢 Security hardening: BOLA fix, real user deletion, defense-in-depth
  (internal service token + service-side permission re-check + NetworkPolicy),
  access-token revocation (jti denylist), constant-time login, per-IP rate
  limiting, body-size limit, startup secret guards.

## ✅ v0.2 — Security+ (shipped)

- 🟢 **Refresh-token reuse detection** → revokes the whole session family. **S**
- 🟢 **Audit log** for sensitive actions, readable at `GET /audit`. **M**
- 🟢 **Per-account lockout** after N failed logins (configurable). **S**
- 🟢 **Email verification** + **password reset** (single-use tokens; dev returns the token, prod emails it). **M**
- 🟡 **TLS/mTLS** — opt-in cert generator (`scripts/gen-certs.sh`) + docs; wire per deployment. **M**
- 🟡 **Secrets via Vault / Sealed Secrets** — documented opt-in (default uses K8s Secret). **M**

## ✅ v0.4 — True microservices (shipped)

- 🟢 **Separate repository per service** (gateway/auth/user) + shared contracts
  and libs repos; each built, versioned (semver tags) and deployed independently. **L**
- 🟢 **One database instance per service** (`postgres-auth`, `postgres-user`). **S**
- 🟢 **Event-driven** register/delete via a **transactional outbox → NATS
  JetStream → idempotent consumer** (no synchronous cross-service writes). **L**
- 🟢 **CI/CD per repo** (GitHub Actions): build/test + service images published
  to GHCR; the umbrella compose pulls them. **M**

## ✅ v0.5 — Observability (shipped)

- 🟢 **OpenTelemetry** distributed tracing → **Jaeger** (gateway → auth/user → DB,
  including SQL spans via otelpgx). **M**
- 🟢 **Prometheus** metrics (`/metrics`) + **Grafana** "IAM Overview" dashboard
  (latency, RPS, login failures). **M**
- 🟢 **Correlation IDs** (`X-Request-Id`) propagated through gRPC metadata + logs. **S**
- 🟢 **Linked cross-service traces** (Rust upgraded to tonic 0.14 for unbroken
  gateway → service spans). **M**

## ✅ v0.6 — Show it off (shipped)

- 🟢 **OpenAPI 3 spec + Swagger UI** at `/docs` on both gateways (vendored, no CDN);
  Authorize → Bearer and try every endpoint live. **M**
- 🟢 **Live demo via GitOps**: ArgoCD on **k3s** (kustomize overlays + Traefik
  ingress + Cloudflare), both stacks running side by side. **L**
- 🟢 **Load tests** (k6) + published **[BENCHMARKS.md](BENCHMARKS.md)**: fair
  Go-vs-Rust on identical infra (read-path, mixed, rate-limiter on/off). **S**
- 🟢 **Configurable auth rate limit** (`AUTH_RATE_LIMIT` / `AUTH_RATE_WINDOW_SECONDS`,
  `0` disables). **S**
- 🟢 **Self-hosted mirror** of all repos to Gitea. **S**

## ✅ v0.7 — OIDC / OAuth2 provider (shipped, M2)

The flagship "wow": make the IAM a real identity provider apps can integrate with.

- ✅ **Discovery** (`.well-known/openid-configuration`) + **JWKS** endpoint with
  rotating signing keys. **M**
- ✅ **Authorization Code + PKCE** flow: `/authorize`, `/token`, `/userinfo`,
  consent screen. **L**
- ✅ **ID tokens** (OIDC claims) + **client registration** (confidential + public clients). **M**
- ✅ Sample relying-party app demonstrating **"Login with iam"**. **S**
- 🔮 Act as an RP too — **social login** (Google/GitHub). **M**

## ✅ v0.8 — Horizontal scale + Redis (shipped, M3)

Make it genuinely multi-instance (builds on the rate-limiter work in v0.6).

- ✅ **Redis-backed rate limiter** — per-IP fixed window in Redis (atomic
  `INCR`+`EXPIRE`), shared across gateway replicas; in-memory fallback when
  `REDIS_URL` is unset, fail-open on Redis error. **M**
- ✅ **Multi-replica gateways** (replicas=2) behind Traefik; proven the limit is
  enforced globally, not per-pod (5 global vs 5×2: 8 requests → 5 pass, 3×429). **S**
- ✅ **Horizontal-scale section** in BENCHMARKS.md. **S**
- ✅ **Redis token denylist** (shared across replicas) + **Redis permission cache**
  (short TTL, invalidated on role change) — landed in v0.9.1. **S/M**

## ✅ v0.9 — Enterprise auth features (shipped, M4)

- ✅ **2FA / TOTP** — opt-in self-service enroll → activate → disable, with
  one-time **recovery codes**; login becomes a challenge (password → `mfa_token`
  → TOTP/recovery code). **M**
- ✅ **API keys** (`iamk_…`) — **scoped** programmatic credentials (scopes must be
  a subset of the creator's perms; effective scope = requested ∩ current perms),
  stored hashed, expiry + revoke. **M**
- ✅ **Soft-delete + restore** — `deleted_at` on identity + profile; soft-deleted
  users can't log in and are hidden, `?hard=true` for permanent delete. **S**
- ✅ **Bulk operations** — assign one role to many users (`POST /roles/{name}/assignments`), landed in v0.9.1. **S**

## 🔮 Backlog — engineering rigor · M5

- ✅ **Integration tests** with testcontainers (Postgres) for the auth repo — Go (`-tags=integration`) + Rust (`--features integration`), green in CI. **M**
- 🟡 **mTLS** between gateway ↔ services, wired up (generator already ships). **M**
- ✅ **Helm chart** deploying either stack (config/secrets from per-stack values) + root Makefile. **S**
- ✅ Fixed the `ListRoles` N+1 — one `LEFT JOIN + array_agg` query (both stacks). **S**
- ✅ **Permanently-failed profile creation** handled — the user service emits a failure event after N retries; auth records it and the profile self-heals on next read (forward recovery, user stays active). **S**

---

# Roadmap (Bahasa Indonesia)

🌐 [English](#roadmap--iam-go--rust) | **Bahasa Indonesia**

Roadmap ini berlaku untuk **kedua** stack (`iam-go`, `iam-rust`) — perubahan
diterapkan di keduanya agar tetap setara. Keterangan: 🟢 selesai · 🟡 sebagian/opt-in
· 🔜 direncanakan · 🔮 ke depan · effort **S/M/L**.

## ✅ v0.1 — Fondasi & pengerasan (sudah rilis)

- 🟢 Auth service: register, login, JWT access + refresh (rotasi), logout.
- 🟢 User service: CRUD profil + pencarian berpaginasi.
- 🟢 API gateway (REST→gRPC) dengan otorisasi per-route.
- 🟢 RBAC granular: role → permission, dinamis; manajemen role lengkap.
- 🟢 Hash password Argon2id (kedua stack), PostgreSQL, Docker Compose + K8s.
- 🟢 Dokumentasi dwibahasa (EN/ID), koleksi Postman + Bruno, smoke test.
- 🟢 Pengerasan keamanan: fix BOLA, hapus user beneran, pertahanan berlapis
  (token internal antar-service + cek ulang permission di service + NetworkPolicy),
  pencabutan access token (denylist jti), login constant-time, rate-limit per IP,
  batas ukuran body, guard secret saat startup.

## ✅ v0.2 — Keamanan lanjutan (sudah rilis)

- 🟢 **Deteksi reuse refresh-token** → cabut seluruh sesi terkait. **S**
- 🟢 **Audit log** aksi sensitif, dibaca via `GET /audit`. **M**
- 🟢 **Lockout per-akun** setelah N login gagal (bisa dikonfigurasi). **S**
- 🟢 **Verifikasi email** + **reset password** (token sekali pakai; dev mengembalikan token, prod via email). **M**
- 🟡 **TLS/mTLS** — generator sertifikat opt-in (`scripts/gen-certs.sh`) + docs; aktifkan per deployment. **M**
- 🟡 **Secret via Vault / Sealed Secrets** — opt-in terdokumentasi (default pakai K8s Secret). **M**

## ✅ v0.4 — True microservices (sudah rilis)

- 🟢 **Repo terpisah per service** (gateway/auth/user) + repo contracts & libs
  bersama; tiap repo di-build, di-versioning (tag semver), dan di-deploy independen. **L**
- 🟢 **Satu instance database per service** (`postgres-auth`, `postgres-user`). **S**
- 🟢 **Event-driven** register/delete lewat **outbox transaksional → NATS
  JetStream → konsumen idempoten** (tanpa panggilan sinkron antar-service). **L**
- 🟢 **CI/CD per repo** (GitHub Actions): build/test + image service ke GHCR;
  compose umbrella menariknya. **M**

## ✅ v0.5 — Observability (sudah rilis)

- 🟢 **OpenTelemetry** tracing terdistribusi → **Jaeger** (gateway → auth/user → DB,
  termasuk SQL span via otelpgx). **M**
- 🟢 **Prometheus** metrics (`/metrics`) + dashboard **Grafana** "IAM Overview". **M**
- 🟢 **Correlation ID** (`X-Request-Id`) diteruskan lewat metadata gRPC + log. **S**
- 🟢 **Trace antar-service nyambung** (Rust di-upgrade ke tonic 0.14). **M**

## ✅ v0.6 — Show it off (sudah rilis)

- 🟢 **OpenAPI 3 + Swagger UI** di `/docs` (kedua gateway, vendored, tanpa CDN). **M**
- 🟢 **Live demo via GitOps**: ArgoCD di **k3s** (kustomize + Traefik + Cloudflare),
  dua stack jalan berdampingan. **L**
- 🟢 **Load test** (k6) + **[BENCHMARKS.md](BENCHMARKS.md)**: Go-vs-Rust di infra
  identik (read-path, mixed, limiter on/off). **S**
- 🟢 **Rate limit auth bisa dikonfigurasi** (`AUTH_RATE_LIMIT` / `AUTH_RATE_WINDOW_SECONDS`,
  `0` = mati). **S**
- 🟢 **Mirror self-hosted** semua repo ke Gitea. **S**

## ✅ v0.7 — OIDC / OAuth2 provider (sudah rilis, M2)

Fitur unggulan: jadikan IAM sebagai **identity provider** yang bisa diintegrasi app lain.

- ✅ **Discovery** (`.well-known/openid-configuration`) + endpoint **JWKS** dengan
  rotasi kunci. **M**
- ✅ Flow **Authorization Code + PKCE**: `/authorize`, `/token`, `/userinfo`,
  layar consent. **L**
- ✅ **ID token** (klaim OIDC) + **registrasi client** (confidential + public). **M**
- ✅ App relying-party contoh untuk demo **"Login with iam"**. **S**
- 🔮 Jadi RP juga — **social login** (Google/GitHub). **M**

## ✅ v0.8 — Skala horizontal + Redis (sudah rilis, M3)

Bikin benar-benar multi-instance (lanjutan kerja rate-limiter di v0.6).

- ✅ **Rate limiter via Redis** — fixed-window per-IP di Redis (`INCR`+`EXPIRE`
  atomik), dibagi lintas replica gateway; fallback in-memory kalau `REDIS_URL`
  kosong, fail-open saat Redis error. **M**
- ✅ **Gateway multi-replica** (replicas=2) di belakang Traefik; terbukti limit
  ditegakkan global, bukan per-pod (5 global vs 5×2: 8 request → 5 lolos, 3×429). **S**
- ✅ **Section horizontal-scale** di BENCHMARKS.md. **S**
- ✅ **Token denylist** (dibagi antar replica) + **cache permission** via Redis
  (TTL pendek, di-invalidate saat role berubah) — masuk di v0.9.1. **S/M**

## ✅ v0.9 — Fitur enterprise auth (sudah rilis, M4)

- ✅ **2FA / TOTP** — opt-in self-service enroll → activate → disable, dengan
  **recovery code** sekali pakai; login jadi challenge (password → `mfa_token`
  → kode TOTP/recovery). **M**
- ✅ **API key** (`iamk_…`) — kredensial programatik **scoped** (scope wajib
  subset permission pembuat; scope efektif = diminta ∩ permission saat ini),
  disimpan hashed, expiry + revoke. **M**
- ✅ **Soft-delete + restore** — `deleted_at` di identity + profile; user
  ter-soft-delete tak bisa login & disembunyikan, `?hard=true` untuk hapus permanen. **S**
- ✅ **Operasi massal** — assign satu role ke banyak user (`POST /roles/{name}/assignments`), masuk di v0.9.1. **S**

## 🔮 Backlog — engineering rigor · M5

- ✅ **Integration test** dengan testcontainers (Postgres) untuk repo auth — Go + Rust, hijau di CI. **M**
- 🟡 **mTLS** antara gateway ↔ service, di-wire (generator sudah ada). **M**
- ✅ **Helm chart** deploy salah satu stack (config/secret dari values per-stack) + Makefile root. **S**
- ✅ N+1 di `ListRoles` diperbaiki — satu query `LEFT JOIN + array_agg` (dua stack). **S**
- ✅ **Pembuatan profil gagal permanen** ditangani — user-service emit event gagal setelah N retry; auth catat, profile self-heal pas dibaca (forward recovery, user tetap aktif). **S**
