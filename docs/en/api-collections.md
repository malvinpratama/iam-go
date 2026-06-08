# API Collections (Postman & Bruno) — iam-go

🌐 **English** | [Bahasa Indonesia](../id/api-collections.md) · [↑ Docs index](README.md)

Two **native** collections live at the repository root (shared by both stacks —
the REST API is identical):

- **Postman**: `iam.postman_collection.json` + `iam.postman_environment.json`
- **Bruno**: `IAM — User & Auth (Go - Rust)/` (open-collection folder)

> ⚠️ **Do NOT import the Postman JSON into Bruno.** Postman uses the `pm.*` API
> and Bruno uses `bru.*`/`res.*`; Bruno's importer only partially translates
> scripts and leaves `pm.collectionVariables.set(...)`, which fails at runtime
> with `ReferenceError: pm is not defined`. Use the native Bruno collection
> instead (Open Collection).

## Postman / Newman

1. Import both files into Postman.
2. Select the **IAM — Local** environment (optional; the collection also carries
   its own variables).
3. Run **Auth → Login (admin)**, then **Register**, **Login (user)**.
   Login/Register/Refresh post-scripts auto-save `access_token`, `refresh_token`,
   `token_type`, `admin_access_token`, `user_id` into **collection variables**
   (`pm.collectionVariables.set`), so the Users/RBAC requests need no edits.

CLI (Newman):
```bash
npx newman run iam.postman_collection.json -e iam.postman_environment.json
```

## Bruno

1. **Open Collection** → select the folder `IAM — User & Auth (Go - Rust)`.
2. Pick the **IAM — Local** environment (top-right).
3. Same order: Login (admin) → Register → Login (user) → explore.
   Post-scripts use `bru.setVar(...)` so `{{access_token}}` resolves on the next
   request — no need to look at the environment editor (script-set values are
   runtime; view them via the variables 👁 icon, not the editor).

CLI (Bruno):
```bash
cd "IAM — User & Auth (Go - Rust)"
npx @usebruno/cli run --env "IAM — Local" -r
```

## Folders / requests

- **Auth**: Register, Login (user), Login (admin), Refresh, Logout
- **Users**: My Identity, Get My Profile, Get User by ID, List Users, Update Profile, Delete User
- **RBAC**: List Roles, List Permissions, Create/Update/Delete Role, Grant/Revoke Permission to Role, Assign/Revoke Role to User
- **Health**: Healthz

Admin-only requests use `{{admin_access_token}}`; user-scoped requests use
`{{access_token}}`. To see RBAC denial, call **Get User by ID** with the user
token → `403`.

See the [API reference](api-reference.md) for full request/response details.

> ⚠️ **Logout now revokes your access token** (jti denylist), so run **Logout last** — in an automated full-collection run it invalidates subsequent authenticated requests.
