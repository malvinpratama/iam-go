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

- 🔜 **Discovery** (`.well-known/openid-configuration`) + **JWKS** endpoint with
  rotating signing keys. **M**
- 🔜 **Authorization Code + PKCE** flow: `/authorize`, `/token`, `/userinfo`,
  consent screen. **L**
- 🔜 **ID tokens** (OIDC claims) + **client registration** (confidential + public clients). **M**
- 🔜 Sample relying-party app demonstrating **"Login with iam"**. **S**
- 🔮 Act as an RP too — **social login** (Google/GitHub). **M**

## 🔜 v0.8 — Horizontal scale + Redis · M3

Make it genuinely multi-instance (builds on the rate-limiter work in v0.6).

- 🔜 **Redis-backed token denylist** (shared across gateway replicas). **S**
- 🔜 **Redis-backed rate limiter** — shared window, replacing the per-pod
  in-memory limiter (consistent cluster-wide). **M**
- 🔜 **Redis permission cache** (cut per-request RBAC lookups). **M**
- 🔜 **Multi-replica gateways** behind Traefik; prove denylist + limiter stay
  consistent across pods. **S**
- 🔜 **Benchmark at N replicas** → throughput-scaling chart in BENCHMARKS.md. **S**

## 🔮 v0.9 — Enterprise auth features · M4

- 🔮 **2FA / TOTP** (+ recovery codes), required for admins. **M**
- 🔮 **API keys / service accounts** for non-human auth. **M**
- 🔮 **Soft-delete + restore**, and enforce the `status` field (suspend blocks login). **S**
- 🔮 **Bulk operations** (assign role to many users). **S**

## 🔮 Backlog — engineering rigor · M5

- 🔜 **Integration tests** with testcontainers (Postgres) for repos/handlers. **M**
- 🟡 **mTLS** between gateway ↔ services, wired up (generator already ships). **M**
- 🔮 **Helm chart** (alternative to kustomize); root Makefile orchestrating both stacks. **S**
- 🔜 Fix `ListRoles` N+1 query. **S**
- 🔮 **Compensation saga** for permanently-failed profile creation. **S**

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

- 🔜 **Discovery** (`.well-known/openid-configuration`) + endpoint **JWKS** dengan
  rotasi kunci. **M**
- 🔜 Flow **Authorization Code + PKCE**: `/authorize`, `/token`, `/userinfo`,
  layar consent. **L**
- 🔜 **ID token** (klaim OIDC) + **registrasi client** (confidential + public). **M**
- 🔜 App relying-party contoh untuk demo **"Login with iam"**. **S**
- 🔮 Jadi RP juga — **social login** (Google/GitHub). **M**

## 🔜 v0.8 — Skala horizontal + Redis · M3

Bikin benar-benar multi-instance (lanjutan kerja rate-limiter di v0.6).

- 🔜 **Token denylist via Redis** (dibagi antar replica gateway). **S**
- 🔜 **Rate limiter via Redis** — window dibagi, ganti limiter in-memory per-pod
  (konsisten lintas pod). **M**
- 🔜 **Cache permission via Redis** (kurangi lookup RBAC per request). **M**
- 🔜 **Gateway multi-replica** di belakang Traefik; buktikan denylist + limiter konsisten. **S**
- 🔜 **Benchmark di N replica** → grafik scaling throughput di BENCHMARKS.md. **S**

## 🔮 v0.9 — Fitur enterprise auth · M4

- 🔮 **2FA / TOTP** (+ recovery code), wajib untuk admin. **M**
- 🔮 **API key / service account** untuk auth non-manusia. **M**
- 🔮 **Soft-delete + restore**, dan menegakkan field `status` (suspend memblok login). **S**
- 🔮 **Operasi massal** (assign role ke banyak user). **S**

## 🔮 Backlog — engineering rigor · M5

- 🔜 **Integration test** dengan testcontainers (Postgres) — repo + handler. **M**
- 🟡 **mTLS** antara gateway ↔ service, di-wire (generator sudah ada). **M**
- 🔮 **Helm chart** (alternatif kustomize); Makefile root untuk kedua stack. **S**
- 🔜 Perbaiki N+1 di `ListRoles`. **S**
- 🔮 **Saga kompensasi** untuk pembuatan profil yang gagal permanen. **S**
