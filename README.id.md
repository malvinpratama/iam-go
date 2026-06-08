# iam-go

🌐 [English](README.md) | **Bahasa Indonesia**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![gRPC](https://img.shields.io/badge/gRPC-go--grpc-244c5a)](https://grpc.io)
[![PRs welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

**Identity & Access Management** — microservice Auth + User dengan **RBAC
granular**, dibangun dengan **Go**. Implementasi Rust pendamping:
[`../iam-rust`](../iam-rust).

> Stack: **Go · Gin** (REST gateway) · **gRPC** (antar-service) · **PostgreSQL** ·
> **sqlc** · **JWT** (access + refresh, bisa di-revoke).

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
                     └ validasi JWT, resolve permission, enforce RBAC per route
```

Diagram & alur lengkap: **[docs/id/architecture.md](docs/id/architecture.md)**.

## Mulai cepat

```bash
make up        # build + jalankan postgres + auth + user + gateway
make smoke     # smoke test end-to-end ke http://localhost:8080
make down      # hentikan + hapus volume
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

```
proto/        kontrak gRPC            gen/        kode Go ter-generate
pkg/          pustaka bersama         services/   auth · user · gateway
deploy/       compose · k8s           scripts/    smoke.sh
docs/         en/ · id/ (dwibahasa)
```

## Dokumentasi

Dokumentasi lengkap dwibahasa di **[`docs/`](docs/id/README.md)**: Arsitektur ·
API Reference · RBAC · Deployment · Development (dengan ERD DB) · API Collections.

## Pengembangan

```bash
make tools     # pasang buf + plugin protoc + sqlc (sekali)
make proto     # regen stub gRPC
make sqlc      # regen kode akses DB
make test      # unit test
```

Detail: **[docs/id/development.md](docs/id/development.md)**.

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
