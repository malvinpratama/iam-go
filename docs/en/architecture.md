# Architecture — iam-go

🌐 **English** | [Bahasa Indonesia](../id/architecture.md) · [↑ Docs index](README.md)

## Overview

`iam-go` is an Identity & Access Management system split into three services:

- **Auth Service** (gRPC) — source of truth for identity, credentials, JWT
  tokens, and all RBAC (roles, permissions, assignments).
- **User Service** (gRPC) — user profiles, keyed by the canonical `user_id`
  minted by Auth.
- **API Gateway** (Gin, REST) — the only public entrypoint. Validates the JWT,
  resolves the caller's permissions, enforces RBAC per route, and translates
  REST → gRPC.

PostgreSQL is the datastore (a logical database per service: `auth_db`,
`user_db`).

## Component diagram

```mermaid
flowchart LR
    C[Client<br/>curl / Postman / Bruno] -->|REST/JSON| GW[API Gateway<br/>Gin]
    GW -->|gRPC| AUTH[Auth Service]
    GW -->|gRPC| USER[User Service]
    AUTH --> ADB[(Postgres<br/>auth_db)]
    USER --> UDB[(Postgres<br/>user_db)]
    GW -.->|ValidateToken| AUTH
```

## Responsibilities

| Service | Owns | Key operations |
|---|---|---|
| Auth | `users`, `refresh_tokens`, `roles`, `permissions`, `role_permissions`, `user_roles` | Register, Login, Refresh, Logout, ValidateToken, RBAC management |
| User | `profiles` | CreateProfile, GetProfile, UpdateProfile, DeleteProfile, ListProfiles |
| Gateway | nothing (stateless) | AuthN (JWT), AuthZ (permission check), REST↔gRPC, register orchestration |

The internal services trust the identity the gateway puts in gRPC metadata
(`x-user-id`, `x-user-email`, `x-user-roles`, `x-user-permissions`) because only
the gateway is reachable from outside; the services sit on the internal network.

## Flow: login + authenticated request

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

## Flow: registration (gateway orchestration)

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

## Tokens

- **Access token**: short-lived JWT (HS256), carries `sub` (user_id) + `email`.
  Permissions are NOT baked into the token — they are resolved fresh from the DB
  on every `ValidateToken` call, so role changes take effect immediately
  (dynamic RBAC).
- **Refresh token**: long-lived opaque random string. Only its SHA-256 hash is
  stored; it can be revoked (logout) and is rotated on every refresh.

See also: [RBAC model](rbac.md) · [API reference](api-reference.md) ·
[Development](development.md) for the database ERD.
