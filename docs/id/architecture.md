# Arsitektur — iam-go

🌐 [English](../en/architecture.md) | **Bahasa Indonesia** · [↑ Indeks dokumentasi](README.md)

## Ikhtisar

`iam-go` adalah sistem Identity & Access Management yang dipecah menjadi tiga layanan:

- **Auth Service** (gRPC) — sumber kebenaran untuk identitas, kredensial, token
  JWT, dan seluruh RBAC (role, permission, penugasan).
- **User Service** (gRPC) — profil pengguna, dikunci dengan `user_id` kanonik
  yang dihasilkan oleh Auth.
- **API Gateway** (Gin, REST) — satu-satunya titik masuk publik. Memvalidasi JWT,
  menyelesaikan permission pemanggil, menegakkan RBAC per route, dan menerjemahkan
  REST → gRPC.

PostgreSQL adalah datastore-nya (satu database logis per layanan: `auth_db`,
`user_db`).

## Diagram komponen

```mermaid
flowchart LR
    C[Client<br/>curl / Postman / Bruno] -->|REST/JSON| GW[API Gateway<br/>Gin]
    GW -->|gRPC| AUTH[Auth Service]
    GW -->|gRPC| USER[User Service]
    AUTH --> ADB[(Postgres<br/>auth_db)]
    USER --> UDB[(Postgres<br/>user_db)]
    GW -.->|ValidateToken| AUTH
```

## Tanggung jawab

| Layanan | Memiliki | Operasi utama |
|---|---|---|
| Auth | `users`, `refresh_tokens`, `roles`, `permissions`, `role_permissions`, `user_roles` | Register, Login, Refresh, Logout, ValidateToken, manajemen RBAC |
| User | `profiles` | CreateProfile, GetProfile, UpdateProfile, DeleteProfile, ListProfiles |
| Gateway | tidak ada (stateless) | AuthN (JWT), AuthZ (pengecekan permission), REST↔gRPC, orkestrasi register |

Layanan internal mempercayai identitas yang diletakkan gateway dalam metadata gRPC
(`x-user-id`, `x-user-email`, `x-user-roles`, `x-user-permissions`) karena hanya
gateway yang dapat dijangkau dari luar; layanan-layanan berada di jaringan internal.

## Alur: login + request terautentikasi

```mermaid
sequenceDiagram
    participant C as Client
    participant GW as Gateway
    participant A as Auth
    participant U as User
    C->>GW: POST /auth/login {email, password}
    GW->>A: Login()
    A->>A: verify password (argon2), issue JWT + refresh
    A-->>GW: access_token, refresh_token
    GW-->>C: 200 token pair
    C->>GW: GET /users/:id (Bearer access_token)
    GW->>A: ValidateToken(access_token)
    A-->>GW: user_id, roles, permissions
    GW->>GW: require "user:read"?
    alt has permission
        GW->>U: GetProfile() + identity metadata
        U-->>GW: profile
        GW-->>C: 200 profile
    else missing permission
        GW-->>C: 403 Forbidden
    end
```

## Alur: registrasi (orkestrasi gateway)

```mermaid
sequenceDiagram
    participant C as Client
    participant GW as Gateway
    participant A as Auth
    participant U as User
    C->>GW: POST /auth/register {email, password}
    GW->>A: Register() → mint user_id, assign role "user"
    A-->>GW: user_id
    GW->>U: CreateProfile(user_id, display_name)
    U-->>GW: profile
    GW-->>C: 201 {user_id, email}
```

## Token

- **Access token**: JWT berumur pendek (HS256), membawa `sub` (user_id) + `email`.
  Permission TIDAK ditanamkan ke dalam token — permission diselesaikan secara segar
  dari DB pada setiap panggilan `ValidateToken`, sehingga perubahan role berlaku
  seketika (RBAC dinamis).
- **Refresh token**: string acak opak berumur panjang. Hanya hash SHA-256-nya yang
  disimpan; token ini dapat dicabut (logout) dan dirotasi pada setiap refresh.

Lihat juga: [Model RBAC](rbac.md) · [Referensi API](api-reference.md) ·
[Pengembangan](development.md) untuk ERD database.
