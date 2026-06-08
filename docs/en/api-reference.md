# API Reference — iam-go

🌐 **English** | [Bahasa Indonesia](../id/api-reference.md) · [↑ Docs index](README.md)

Base URL (local): `http://localhost:8080`. All bodies are JSON. Authenticated
requests use `Authorization: Bearer <access_token>`.

## Endpoint summary

| Method | Path | Permission |
|---|---|---|
| POST | `/auth/register` | public |
| POST | `/auth/login` | public |
| POST | `/auth/refresh` | public (valid refresh token) |
| POST | `/auth/logout` | authenticated |
| GET | `/me` | authenticated (your roles & permissions) |
| GET | `/users/me` | authenticated (your profile) |
| GET | `/users/:id` | `user:read` |
| GET | `/users` | `user:read` |
| PATCH | `/users/:id` | self, or `user:write` (other users) |
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
| GET | `/healthz` | public |

## Error model

Errors return `{ "error": "message" }` with these status codes:

| HTTP | When |
|---|---|
| 400 | invalid input / validation |
| 401 | missing/invalid/expired token, bad credentials, revoked refresh |
| 403 | authenticated but missing the required permission |
| 404 | resource not found |
| 409 | conflict (e.g. email already registered, deleting a built-in role) |

---

## Auth

### POST /auth/register — public
```json
{ "email": "alice@iam.local", "password": "alicepass123" }
```
`201` → `{ "user_id": "uuid", "email": "alice@iam.local" }`. Creates the user
(default role `user`) and its profile. `409` if email exists.

### POST /auth/login — public
```json
{ "email": "alice@iam.local", "password": "alicepass123" }
```
`200` →
```json
{ "access_token": "eyJ...", "refresh_token": "hex...", "expires_in": 900, "token_type": "Bearer" }
```
`401` on bad credentials.

### POST /auth/refresh — public (valid refresh token)
```json
{ "refresh_token": "hex..." }
```
`200` → new token pair (the old refresh token is revoked — rotation).
`401` if the token is invalid, expired, or revoked.

### POST /auth/logout — authenticated
```json
{ "refresh_token": "hex..." }
```
`200` → `{ "success": true }`. Revokes the refresh token.

---

## Identity & users

### GET /me — authenticated
`200` →
```json
{ "user_id": "uuid", "email": "alice@iam.local", "roles": ["user"], "permissions": ["profile:read","profile:write"] }
```
Your own identity. Use it to discover what you can do.

### GET /users/me — authenticated
`200` → your profile (`user_id, display_name, bio, avatar_url, phone, created_at, updated_at`).

### GET /users/:id — `user:read`
`200` → the profile of `:id`. `403` without `user:read`. `404` if absent.

### GET /users — `user:read`
Query: `?page=1&page_size=20&query=<search display_name>`.
`200` → `{ "profiles": [...], "total": N, "page": 1, "page_size": 20 }`.

### PATCH /users/:id — self, or `user:write`
```json
{ "display_name": "Alice", "bio": "...", "avatar_url": "...", "phone": "..." }
```
All fields optional (sparse update). `200` → updated profile. `403` if updating
someone else without `profile:write`.

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
`201` → the created role. `409` if it exists.

### PATCH /roles/:name — `role:write`
```json
{ "description": "Updated description" }
```
`200` → updated role. `404` if not found.

### DELETE /roles/:name — `role:write`
`200` → `{ "success": true }`. `409` for built-in roles (`admin`, `user`).
`404` if not found.

### POST /roles/:name/permissions — `role:write`
```json
{ "permission": "user:read" }
```
`200` → `{ "success": true }`. Grants a permission to the role.

### DELETE /roles/:name/permissions/:perm — `role:write`
`200` → `{ "success": true }`. Revokes a permission from the role.

### POST /users/:id/roles — `role:assign`
```json
{ "role": "admin" }
```
`200` → `{ "success": true }`. Assigns a role to the user.

### DELETE /users/:id/roles/:role — `role:assign`
`200` → `{ "success": true }`. Removes a role from the user.

---

## gRPC contracts (internal)

Defined in `proto/auth/v1/auth.proto` and `proto/user/v1/user.proto`.

- **AuthService**: `Register`, `Login`, `Refresh`, `Logout`, `ValidateToken`,
  `CreateRole`, `UpdateRole`, `DeleteRole`, `ListRoles`, `AssignRole`,
  `RevokeRole`, `ListPermissions`, `GrantPermission`, `RevokePermission`.
- **UserService**: `CreateProfile`, `GetProfile`, `UpdateProfile`,
  `DeleteProfile`, `ListProfiles`.

`ValidateToken` returns `user_id, email, roles[], permissions[]` and is called by
the gateway on every authenticated request.
---

## Security notes

- **`/auth/*` is rate-limited per IP** (HTTP 429 when exceeded).
- **Logout revokes the access token**, not just the refresh token: the access
  token's `jti` is added to a denylist checked on every request, so it stops
  working immediately (no ~15-minute window).
- **PATCH `/users/:id`**: editing your OWN profile needs `profile:write`; editing
  ANOTHER user's profile needs `user:write` (admin) — otherwise `403`.
- **DELETE `/users/:id`** removes the **identity** (credentials, roles, refresh
  tokens) AND the profile; the user can no longer log in afterward.
- **Defense in depth**: the internal gRPC services independently re-check the
  required permission and require a shared `INTERNAL_SERVICE_TOKEN` from the
  gateway, so they reject any caller that is not the gateway.
