# iam-go

🌐 [English](README.md) | **Bahasa Indonesia**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![gRPC](https://img.shields.io/badge/gRPC-go--grpc-244c5a)](https://grpc.io)
[![PRs welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

**Identity & Access Management** — microservice Auth + User dengan **RBAC
granular**, dibangun dengan **Go**. Ini repo **platform/umbrella**: meng-orkestrasi
service yang di-deploy independen, serta memuat deployment, dokumentasi, dan
koleksi API. Implementasi Rust pendamping:
[iam-rust](https://github.com/malvinpratama/iam-rust).

> Stack: **Go · Gin** (REST gateway) · **gRPC** (antar-service) · **NATS JetStream**
> (event async) · **PostgreSQL** (satu DB per service) · **sqlc** · **JWT**
> (access + refresh, bisa di-revoke).

## Repositori

Tiap service adalah repo tersendiri — di-build, di-versioning, dan di-deploy
independen; kode bersama ada di repo modul khusus.

| Repo | Peran |
|---|---|
| [iam-go-gateway](https://github.com/malvinpratama/iam-go-gateway) | API gateway REST→gRPC, otorisasi per-route |
| [iam-go-auth](https://github.com/malvinpratama/iam-go-auth) | Service Auth + RBAC (pemilik `auth_db`, penerbit event) |
| [iam-go-user](https://github.com/malvinpratama/iam-go-user) | Service profil (pemilik `user_db`, konsumen event) |
| [iam-go-contracts](https://github.com/malvinpratama/iam-go-contracts) | `.proto` bersama + stub gRPC ter-generate |
| [iam-go-libs](https://github.com/malvinpratama/iam-go-libs) | Pustaka bersama (config, db, jwt, NATS, …) |
| **iam-go** (repo ini) | Platform: compose · k8s · docs · koleksi · smoke |

## Fitur

- 🔐 **Auth**: register, login, JWT access + refresh token dengan **rotasi** dan **revocation**.
- 👤 **Users**: CRUD profil + pencarian berpaginasi, lewat service tersendiri.
- 🛡️ **RBAC granular**: role → permission; **dinamis** (perubahan role berlaku di request berikutnya).
- 🧩 **Manajemen role**: buat/ubah/hapus role, grant/revoke permission, assign/revoke role.
- 🚪 **API Gateway**: satu pintu masuk publik, REST→gRPC, otorisasi per-route.
- 📦 **Siap jalan**: Docker Compose + manifest Kubernetes, migrasi & seed otomatis, bootstrap admin.
- ✅ **Terverifikasi**: smoke test end-to-end + koleksi Postman/Bruno.

## Arsitektur

```
client ──REST──▶ Gateway (Gin) ──gRPC──▶ Auth Service ──▶ Postgres (auth_db)
                     │            └─gRPC──▶ User Service ──▶ Postgres (user_db)
                     │                          ▲
                     │   register / delete      │ konsumsi
                     └ validasi JWT, RBAC       │
                                                │
        Auth ──outbox──▶ NATS JetStream ──iam.user.*──┘   (async, eventually consistent)
```

Auth dan User tidak saling memanggil: efek lintas-service (buat profil saat
register, hapus profil saat delete) mengalir lewat **outbox transaksional →
NATS JetStream → konsumen idempoten**. Diagram & alur lengkap:
**[docs/id/architecture.md](docs/id/architecture.md)**.

## Mulai cepat

```bash
make up                 # tarik image service dari GHCR + jalankan stack
make up IMAGE_TAG=dev   # atau pakai image hasil build lokal
make smoke              # smoke test end-to-end ke http://localhost:8080
make down               # hentikan + hapus volume
```

Bootstrap admin (`admin@iam.local` / `admin12345`) dibuat saat pertama boot. Lalu:

```bash
# register, login, dan lihat role & permission milikmu
curl -s localhost:8080/auth/register -H 'Content-Type: application/json' \
  -d '{"email":"alice@iam.local","password":"alicepass123"}'
TOKEN=$(curl -s localhost:8080/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"alice@iam.local","password":"alicepass123"}' | jq -r .access_token)
curl -s localhost:8080/me -H "Authorization: Bearer $TOKEN"
```

## API

REST di `:8080`. Sorotan: `/auth/{register,login,refresh,logout}`, `/me`,
`/users[/:id]`, `/roles`, `/permissions`, manajemen role/permission. Referensi
lengkap dengan contoh & kode error:
**[docs/id/api-reference.md](docs/id/api-reference.md)**.

Coba lewat Postman atau Bruno — lihat
**[docs/id/api-collections.md](docs/id/api-collections.md)**.

## Struktur project

Repo umbrella ini hanya memuat lapisan platform; kode service ada di
[repo per-service](#repositori).

```
deploy/       docker-compose · k8s · .env.example
docs/         en/ · id/ (dwibahasa)
scripts/      smoke.sh
*.json        koleksi Postman + environment
```

## Dokumentasi

Dokumentasi lengkap dwibahasa di **[`docs/`](docs/id/README.md)**: Arsitektur ·
API Reference · RBAC · Deployment · Development (dengan ERD DB) · API Collections.

## Pengembangan

Tiap service dikembangkan di repo masing-masing (`make build` / `make test` /
`make docker` di sana). Untuk kerja lintas-repo, checkout repo berdampingan dan
gabungkan dengan `go.work` (jangan di-commit). Repo contracts & libs di-tag;
service mem-pin versi yang pasti. Detail:
**[docs/id/development.md](docs/id/development.md)**.

## Deployment

Docker Compose (lokal) dan Kubernetes (kustomize) — lihat
**[docs/id/deployment.md](docs/id/deployment.md)**.

## Roadmap

- [ ] Rate limiting di endpoint auth
- [ ] Audit log untuk perubahan RBAC
- [ ] Generasi spec OpenAPI/Swagger
- [ ] Deteksi penggunaan ulang refresh-token

## Berkontribusi

Kontribusi sangat diterima! Lihat **[CONTRIBUTING.md](CONTRIBUTING.md)** dan
**[Code of Conduct](CODE_OF_CONDUCT.md)**.

## Lisensi

[MIT](LICENSE) © 2026 malvin
