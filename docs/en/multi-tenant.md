# Multi-tenancy (Organizations)

🌐 **English** | [Bahasa Indonesia](../id/multi-tenant.md) · [← Docs](README.md)

From **v0.10** the IAM is a **B2B identity platform** (Auth0 / WorkOS / Clerk
"Organizations" style): one global user can belong to many **tenants**
(organizations), each with its own roles, OAuth clients, members and projects.
The same change lands in both stacks (`iam-go`, `iam-rust`) to keep parity.

---

## 1. Model

| Concept | What it is |
|---|---|
| **Tenant** | An isolated organization. Owns roles, projects, members, OAuth clients, API keys. |
| **Project** | A scope **within** a tenant (e.g. `prod`, `staging`). A role can be granted tenant-wide or for one project. |
| **Membership** | A user ↔ tenant link (`memberships` table). A user is a member of N tenants. |
| **User / Profile** | **Global** — one identity (email is globally unique) shared across all tenants (GitHub/Slack style). |

What is **global** vs **per-tenant**:

- **Global:** `users`, `profiles`, the **permission catalog** (verbs are
  product-wide), credentials/crypto artifacts (recovery codes, signing keys,
  OAuth authorization codes…).
- **Per-tenant:** `roles` (built-in `admin`/`user` stay `tenant_id = NULL`
  templates; custom roles belong to a tenant), `user_roles` (scoped by
  `tenant_id` + nullable `project_id`), `oauth_clients`, `api_keys`,
  `refresh_tokens`, and per-tenant `memberships`/`projects`.

A fixed **default tenant** (`00000000-0000-0000-0000-000000000001`) is seeded so
every pre-existing user/role/client is backfilled into it — single-tenant
deployments keep working unchanged.

---

## 2. Tenant-bound tokens

The access token carries the active tenant (and optional project):

```jsonc
// access-token claims
{ "sub": "…", "email": "…", "tenant_id": "…", "project_id": "…", "exp": … }
```

- **Login** authenticates the identity, then binds the token to the user's first
  active membership (default-tenant fallback). The binding is **persisted on the
  `refresh_tokens` row**, so a later **refresh keeps the same tenant/project**.
- **`POST /auth/switch`** re-issues a fresh token pair bound to another tenant
  the caller belongs to (returns `403` if they are not a member). The old token
  is **not** revoked → concurrent sessions in different tenants are fine.
- **`ValidateToken`** (run by the gateway on every request) verifies the user is
  still an **active member** of the token's tenant — removing a membership
  invalidates their tokens for it on the next call.
- The gateway forwards the active tenant to internal services as
  `x-tenant-id` / `x-project-id` metadata.

---

## 3. Scoped RBAC

Roles and permissions resolve **per the token's tenant + project**: a tenant-wide
assignment (`project_id IS NULL`) always applies; a project-scoped one applies
only when the token names that project.

```sql
WHERE ur.user_id = $1
  AND ur.tenant_id = $2
  AND (ur.project_id IS NULL OR ur.project_id = $3)
```

So **the same user can be an admin in one tenant and a plain user in another**.
The permission cache is keyed `perms:{tenant}:{project}:{user}`, and a role
change clears **all** of a user's entries (across every tenant).

**Assigning** a role is likewise scoped: an assignment is written for the active
tenant and an optional project — `project_id` empty = **tenant-wide** (applies to
every project), set = scoped to **that project** only. `GET /users/:id/roles`
lists a user's assignments (role + scope) so an admin can revoke precisely.

---

## 4. Row-Level Security (defense in depth)

Tenant isolation is enforced in **two layers**: the app-layer `WHERE tenant_id`
**and** Postgres **Row-Level Security**.

- The 9 tenant-scoped tables have a `tenant_isolation` policy (`ENABLE` +
  `FORCE`). From v0.10 it is **fail-closed**: a query only sees rows whose
  `tenant_id` matches `app.tenant_id` (plus `NULL`-tenant templates).
- The app connects as a **superuser**, which *bypasses* RLS — so RLS only
  applies to queries wrapped in `WithTenant`: a transaction that runs
  `SET LOCAL ROLE iam_rls` (a non-superuser role) + `set_config('app.tenant_id', …)`
  before the query. There a **forgotten `WHERE tenant_id` still cannot leak**
  another tenant's rows.

```text
iam_rls + app.tenant_id = A → SELECT * FROM projects  (no WHERE) → only tenant A's rows
iam_rls + app.tenant_id unset → 0 tenant rows (fail-closed)
superuser (unwrapped path)    → all rows (RLS bypassed; relies on app-layer WHERE)
```

---

## 5. OIDC: client → tenant

Each OAuth client belongs to a tenant (the organization its app serves):

- `ExchangeAuthorizationCode` binds the session to the **client's tenant** (not
  the user's first tenant) and requires the user to be an active member —
  otherwise the login is denied. So logging in through an org's OIDC client
  yields a token scoped to **that** org.
- `RegisterClient` stamps the new client with the caller's active tenant.

---

## 6. Endpoints

| Method & path | Permission | Purpose |
|---|---|---|
| `GET /me/memberships` | (any authed) | Tenants the caller belongs to (drives the switcher) |
| `POST /auth/switch` | (member) | Re-issue a token bound to another tenant/project |
| `POST /tenants` | `tenant:write` | Create a tenant (creator becomes its admin) |
| `GET /tenants` | `tenant:read` | List tenants |
| `POST /projects` | `project:write` | Create a project in the active tenant |
| `GET /projects` | `project:read` | List the active tenant's projects |
| `GET /members` | `member:read` | List the active tenant's members |
| `POST /members` | `member:write` | Add a member by email |
| `DELETE /members/:userId` | `member:write` | Remove a member |
| `GET /users` | `user:read` | Active-tenant directory (members ⋈ profiles, one batch fetch) |
| `GET /users/:id/roles` | `role:read` | A user's role assignments (role + project scope) in the tenant |
| `POST /users/:id/roles` | `role:assign` | Assign a role (body `project_id` empty = tenant-wide) |
| `DELETE /users/:id/roles/:role` | `role:assign` | Revoke an assignment (`?project_id=` selects the scoped one) |

Creating a tenant runs in one transaction: create tenant → enroll the creator →
grant them the `admin` role **scoped to the new tenant** (platform roles do not
carry across tenants).

---

## 7. Console

The admin console (`iam-console`) gets a **tenant switcher** in the sidebar (fed
by `GET /me/memberships`; selecting one calls `POST /auth/switch` and adopts the
re-issued token into the NextAuth session), plus **Tenants / Projects / Members**
pages and a tenant-scoped **Users** directory — each gated by the matching
permission.

---

## 8. Operational notes

- **No new env/secret** is required; migrations (`000010`–`000013`) run on auth
  startup. Roll out auth (it applies them) then user + gateway.
- **Refresh-token rotation grace** (v0.10): a rotated refresh token re-presented
  within a short grace window is re-issued instead of triggering the
  theft-detection family-wipe — this fixes a session lockout under the
  **concurrent refreshes** an OIDC client (e.g. NextAuth) fires when the access
  token expires. Logout-revoked tokens are still hard-rejected.
