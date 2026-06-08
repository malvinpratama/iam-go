# Dokumentasi iam-go

🌐 [English](../en/README.md) | **Bahasa Indonesia** · [← README Proyek](../../README.md)

Identity & Access Management — **microservice Auth + User dengan RBAC granular**,
dibangun dengan **Go** (Gin · gRPC · sqlc · PostgreSQL · JWT).

## Daftar isi

| Dokumen | Isi |
|---|---|
| [Arsitektur](architecture.md) | Layanan, diagram komponen & sekuens, model token |
| [Referensi API](api-reference.md) | Setiap endpoint REST (request/response/error) + kontrak gRPC |
| [Model RBAC](rbac.md) | Role, permission, seed, RBAC dinamis, manajemen role |
| [Deployment & Ops](deployment.md) | Docker Compose, Kubernetes, env var, migrasi, pemecahan masalah |
| [Pengembangan](development.md) | Toolchain, codegen, struktur proyek, tes, **ERD DB** |
| [Koleksi API](api-collections.md) | Penggunaan Postman & Bruno (dua koleksi native) |

## Mulai cepat

```bash
make up        # build + run the full stack (postgres + auth + user + gateway)
make smoke     # end-to-end smoke test against http://localhost:8080
make down      # stop + remove volumes
```

Seorang admin bootstrap (`admin@iam.local` / `admin12345`) dibuat saat boot pertama.
Daftarkan seorang pengguna, login, lalu panggil `GET /me` untuk melihat role & permission Anda.

## Sekilas

```
client ──REST──▶ Gateway (Gin) ──gRPC──▶ Auth Service ──▶ Postgres (auth_db)
                     │            └─gRPC──▶ User Service ──▶ Postgres (user_db)
                     └ validates JWT, resolves permissions, enforces RBAC per route
```

Implementasi Rust paralel berada di iam-rust (https://github.com/malvinpratama/iam-rust).
