# Contributing to iam-go

🌐 **English** | [Bahasa Indonesia](#berkontribusi-bahasa-indonesia)

Thanks for your interest in contributing! This document explains how to set up
the project, the conventions we follow, and how to submit changes.

## Prerequisites

- Go 1.26+
- Docker + Docker Compose (for the full stack)
- Code generation tools (installed via `make tools`): `buf`, `protoc-gen-go`,
  `protoc-gen-go-grpc`, `sqlc`

## Local setup

```bash
make tools          # one-time: install buf + protoc plugins + sqlc
make proto          # regenerate gRPC stubs from proto/ into gen/
make sqlc           # regenerate DB access code from SQL
make build          # go build ./...
make test           # unit tests
make up             # run the full stack via docker-compose
make smoke          # end-to-end smoke test
make down           # stop + remove volumes
```

## Making changes

1. **Branch** from `main`: `git checkout -b feat/short-description`.
2. If you change `proto/**` → run `make proto`. If you change `db/queries/**`
   or migrations → run `make sqlc`.
3. Keep code idiomatic; match the surrounding style. Run `gofmt`/`go vet`.
4. Add/adjust tests where it makes sense (`make test` must pass).
5. Verify end-to-end: `make up && make smoke`.

## Commit messages

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(auth): add RevokeRole RPC and REST route
fix(gateway): map FailedPrecondition to HTTP 409
docs(rbac): document role:write permission
```

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `build`, `ci`.

## Pull requests

- Keep PRs focused and reasonably small.
- Describe **what** and **why**; link any related issue.
- Ensure `make build`, `make test`, and `make smoke` pass.
- Update docs (`docs/en` **and** `docs/id`) when behavior or APIs change.

By contributing you agree your contributions are licensed under the project's
[MIT License](LICENSE).

---

## Berkontribusi (Bahasa Indonesia)

🌐 [English](#contributing-to-iam-go) | **Bahasa Indonesia**

Terima kasih atas minatnya untuk berkontribusi! Dokumen ini menjelaskan cara
menyiapkan project, konvensi yang dipakai, dan cara mengirim perubahan.

### Prasyarat

- Go 1.26+
- Docker + Docker Compose (untuk menjalankan stack penuh)
- Tools codegen (via `make tools`): `buf`, `protoc-gen-go`, `protoc-gen-go-grpc`, `sqlc`

### Penyiapan lokal

```bash
make tools          # sekali saja: pasang buf + plugin protoc + sqlc
make proto          # regen stub gRPC dari proto/ ke gen/
make sqlc           # regen kode akses DB dari SQL
make build          # go build ./...
make test           # unit test
make up             # jalankan stack via docker-compose
make smoke          # smoke test end-to-end
make down           # hentikan + hapus volume
```

### Membuat perubahan

1. **Branch** dari `main`: `git checkout -b feat/deskripsi-singkat`.
2. Jika mengubah `proto/**` → jalankan `make proto`. Jika mengubah
   `db/queries/**` atau migrasi → jalankan `make sqlc`.
3. Jaga kode tetap idiomatik; ikuti gaya sekitarnya. Jalankan `gofmt`/`go vet`.
4. Tambah/sesuaikan test bila perlu (`make test` harus lulus).
5. Verifikasi end-to-end: `make up && make smoke`.

### Pesan commit

Mengikuti [Conventional Commits](https://www.conventionalcommits.org/). Tipe:
`feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `build`, `ci`.

### Pull request

- Fokus dan tidak terlalu besar.
- Jelaskan **apa** dan **kenapa**; tautkan issue terkait.
- Pastikan `make build`, `make test`, dan `make smoke` lulus.
- Perbarui dokumentasi (`docs/en` **dan** `docs/id`) saat perilaku/API berubah.

Dengan berkontribusi, kamu setuju kontribusimu dilisensikan di bawah
[Lisensi MIT](LICENSE) project ini.
