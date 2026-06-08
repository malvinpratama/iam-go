# Referensi API — iam-go

🌐 [English](../en/api-reference.md) | **Bahasa Indonesia** · [↑ Indeks dokumentasi](README.md)

Base URL (lokal): `http://localhost:8080`. Semua body berformat JSON. Request
terautentikasi menggunakan `Authorization: Bearer <access_token>`.

## Ringkasan endpoint

| Method | Path | Permission |
|---|---|---|
| POST | `/auth/register` | publik |
| POST | `/auth/login` | publik |
| POST | `/auth/refresh` | publik (refresh token valid) |
| POST | `/auth/logout` | terautentikasi |
| GET | `/me` | terautentikasi (role & permission Anda) |
| GET | `/users/me` | terautentikasi (profil Anda) |
| GET | `/users/:id` | `user:read` |
| GET | `/users` | `user:read` |
| PATCH | `/users/:id` | diri sendiri, atau `user:write` (user lain) |
| DELETE | `/users/:id` | `user:delete` |
| GET | `/roles` | `role:read` |
| POST | `/roles` | `role:write` |
| PATCH | `/roles/:name` | `role:write` |
| DELETE | `/roles/:name` | `role:write` |
| POST | `/roles/:name/permissions` | `role:write` |
| DELETE | `/roles/:name/permissions/:perm` | `role:write` |
| GET | `/permissions` | `role:read` |
| POST | `/users/:id/roles` | `role:assign` |
| DELETE | `/users/:id/roles/:role` | `role:assign` |
| GET | `/healthz` | publik |

## Model error

Error mengembalikan `{ "error": "message" }` dengan kode status berikut:

| HTTP | Kapan |
|---|---|
| 400 | input tidak valid / validasi |
| 401 | token hilang/tidak valid/kedaluwarsa, kredensial salah, refresh dicabut |
| 403 | terautentikasi tetapi tidak memiliki permission yang dibutuhkan |
| 404 | resource tidak ditemukan |
| 409 | konflik (mis. email sudah terdaftar, menghapus role bawaan) |

---

## Auth

### POST /auth/register — publik
```json
{ "email": "alice@iam.local", "password": "alicepass123" }
```
`201` → `{ "user_id": "uuid", "email": "alice@iam.local" }`. Membuat pengguna
(role default `user`) beserta profilnya. `409` jika email sudah ada.

### POST /auth/login — publik
```json
{ "email": "alice@iam.local", "password": "alicepass123" }
```
`200` →
```json
{ "access_token": "eyJ...", "refresh_token": "hex...", "expires_in": 900, "token_type": "Bearer" }
```
`401` jika kredensial salah.

### POST /auth/refresh — publik (refresh token valid)
```json
{ "refresh_token": "hex..." }
```
`200` → pasangan token baru (refresh token lama dicabut — rotasi).
`401` jika token tidak valid, kedaluwarsa, atau dicabut.

### POST /auth/logout — terautentikasi
```json
{ "refresh_token": "hex..." }
```
`200` → `{ "success": true }`. Mencabut refresh token.

---

## Identitas & pengguna

### GET /me — terautentikasi
`200` →
```json
{ "user_id": "uuid", "email": "alice@iam.local", "roles": ["user"], "permissions": ["profile:read","profile:write"] }
```
Identitas Anda sendiri. Gunakan untuk mengetahui apa saja yang dapat Anda lakukan.

### GET /users/me — terautentikasi
`200` → profil Anda (`user_id, display_name, bio, avatar_url, phone, created_at, updated_at`).

### GET /users/:id — `user:read`
`200` → profil dari `:id`. `403` tanpa `user:read`. `404` jika tidak ada.

### GET /users — `user:read`
Query: `?page=1&page_size=20&query=<search display_name>`.
`200` → `{ "profiles": [...], "total": N, "page": 1, "page_size": 20 }`.

### PATCH /users/:id — diri sendiri, atau `user:write`
```json
{ "display_name": "Alice", "bio": "...", "avatar_url": "...", "phone": "..." }
```
Semua field opsional (sparse update). `200` → profil yang diperbarui. `403` jika
memperbarui orang lain tanpa `profile:write`.

### DELETE /users/:id — `user:delete`
`200` → `{ "success": true }`.

---

## RBAC

### GET /roles — `role:read`
`200` → `{ "roles": [ { "id", "name", "description", "permissions": [...] } ] }`.

### GET /permissions — `role:read`
`200` → `{ "permissions": [ { "id", "name", "description" } ] }`.

### POST /roles — `role:write`
```json
{ "name": "moderator", "description": "Moderator role" }
```
`201` → role yang dibuat. `409` jika sudah ada.

### PATCH /roles/:name — `role:write`
```json
{ "description": "Updated description" }
```
`200` → role yang diperbarui. `404` jika tidak ditemukan.

### DELETE /roles/:name — `role:write`
`200` → `{ "success": true }`. `409` untuk role bawaan (`admin`, `user`).
`404` jika tidak ditemukan.

### POST /roles/:name/permissions — `role:write`
```json
{ "permission": "user:read" }
```
`200` → `{ "success": true }`. Memberikan sebuah permission ke role.

### DELETE /roles/:name/permissions/:perm — `role:write`
`200` → `{ "success": true }`. Mencabut sebuah permission dari role.

### POST /users/:id/roles — `role:assign`
```json
{ "role": "admin" }
```
`200` → `{ "success": true }`. Menugaskan sebuah role ke pengguna.

### DELETE /users/:id/roles/:role — `role:assign`
`200` → `{ "success": true }`. Menghapus sebuah role dari pengguna.

---

## Kontrak gRPC (internal)

Didefinisikan di `proto/auth/v1/auth.proto` dan `proto/user/v1/user.proto`.

- **AuthService**: `Register`, `Login`, `Refresh`, `Logout`, `ValidateToken`,
  `CreateRole`, `UpdateRole`, `DeleteRole`, `ListRoles`, `AssignRole`,
  `RevokeRole`, `ListPermissions`, `GrantPermission`, `RevokePermission`.
- **UserService**: `CreateProfile`, `GetProfile`, `UpdateProfile`,
  `DeleteProfile`, `ListProfiles`.

`ValidateToken` mengembalikan `user_id, email, roles[], permissions[]` dan dipanggil
oleh gateway pada setiap request terautentikasi.
---

## Catatan keamanan

- **`/auth/*` dibatasi rate per IP** (HTTP 429 bila terlampaui).
- **Logout mencabut access token**, bukan hanya refresh token: `jti` access token
  dimasukkan ke denylist yang dicek tiap request, sehingga langsung tidak berlaku
  (tanpa jeda ~15 menit).
- **PATCH `/users/:id`**: mengubah profil SENDIRI butuh `profile:write`; mengubah
  profil USER LAIN butuh `user:write` (admin) — selain itu `403`.
- **DELETE `/users/:id`** menghapus **identitas** (kredensial, role, refresh
  token) DAN profil; user tidak bisa login lagi setelahnya.
- **Pertahanan berlapis**: service gRPC internal ikut mengecek ulang permission
  yang dibutuhkan dan mewajibkan `INTERNAL_SERVICE_TOKEN` bersama dari gateway,
  sehingga menolak pemanggil yang bukan gateway.
