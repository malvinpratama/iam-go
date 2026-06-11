# Multi-tenant (Organizations)

🌐 [English](../en/multi-tenant.md) | **Bahasa Indonesia** · [← Dokumentasi](README.md)

Mulai **v0.10**, IAM ini jadi **platform identity B2B** (gaya "Organizations" ala
Auth0 / WorkOS / Clerk): satu user global bisa jadi anggota banyak **tenant**
(organisasi), masing-masing punya role, OAuth client, member, dan project
sendiri. Perubahan yang sama masuk ke dua stack (`iam-go`, `iam-rust`) demi
paritas.

---

## 1. Model

| Konsep | Penjelasan |
|---|---|
| **Tenant** | Organisasi terisolasi. Punya role, project, member, OAuth client, API key. |
| **Project** | Scope **di dalam** tenant (mis. `prod`, `staging`). Role bisa diberi tenant-wide atau untuk satu project. |
| **Membership** | Relasi user ↔ tenant (tabel `memberships`). Satu user bisa anggota N tenant. |
| **User / Profile** | **Global** — satu identitas (email unik global) dipakai lintas tenant (gaya GitHub/Slack). |

Yang **global** vs **per-tenant**:

- **Global:** `users`, `profiles`, **katalog permission** (verb itu product-wide),
  artefak kredensial/kripto (recovery code, signing key, OAuth code…).
- **Per-tenant:** `roles` (built-in `admin`/`user` tetap `tenant_id = NULL`
  sebagai template; role custom milik tenant), `user_roles` (di-scope oleh
  `tenant_id` + `project_id` nullable), `oauth_clients`, `api_keys`,
  `refresh_tokens`, serta `memberships`/`projects` per-tenant.

Sebuah **default tenant** (`00000000-0000-0000-0000-000000000001`) di-seed supaya
semua user/role/client lama di-backfill ke sana — deployment single-tenant tetap
jalan tanpa perubahan.

---

## 2. Token terikat tenant

Access token membawa tenant aktif (dan project opsional):

```jsonc
{ "sub": "…", "email": "…", "tenant_id": "…", "project_id": "…", "exp": … }
```

- **Login** mengautentikasi identitas, lalu mengikat token ke membership aktif
  pertama (fallback ke default tenant). Ikatan ini **disimpan di baris
  `refresh_tokens`**, jadi **refresh tetap mempertahankan tenant/project**-nya.
- **`POST /auth/switch`** menerbitkan ulang pasangan token untuk tenant lain yang
  jadi anggota si caller (`403` kalau bukan member). Token lama **tidak**
  dicabut → sesi paralel di tenant berbeda aman.
- **`ValidateToken`** (dipanggil gateway tiap request) memastikan user masih
  **anggota aktif** tenant pada token — mencabut membership langsung membuat
  token-nya invalid di panggilan berikutnya.
- Gateway meneruskan tenant aktif ke service internal lewat metadata
  `x-tenant-id` / `x-project-id`.

---

## 3. RBAC ter-scope

Role & permission diresolusi **per tenant + project** pada token: assignment
tenant-wide (`project_id IS NULL`) selalu berlaku; yang project-scoped hanya
berlaku saat token menyebut project itu.

```sql
WHERE ur.user_id = $1
  AND ur.tenant_id = $2
  AND (ur.project_id IS NULL OR ur.project_id = $3)
```

Jadi **user yang sama bisa jadi admin di satu tenant dan user biasa di tenant
lain**. Cache permission di-key `perms:{tenant}:{project}:{user}`, dan perubahan
role menghapus **semua** entri user itu (di seluruh tenant).

---

## 4. Row-Level Security (defense in depth)

Isolasi tenant ditegakkan di **dua lapis**: `WHERE tenant_id` di app **dan**
**Row-Level Security** Postgres.

- 9 tabel ter-scope tenant punya policy `tenant_isolation` (`ENABLE` + `FORCE`).
  Sejak v0.10 ini **fail-closed**: query hanya melihat baris yang `tenant_id`-nya
  cocok dengan `app.tenant_id` (plus template `tenant_id = NULL`).
- App connect sebagai **superuser** yang *bypass* RLS — jadi RLS hanya berlaku
  untuk query yang dibungkus `WithTenant`: transaksi yang menjalankan
  `SET LOCAL ROLE iam_rls` (role non-superuser) + `set_config('app.tenant_id', …)`
  sebelum query. Di situ **`WHERE tenant_id` yang lupa pun tetap tidak bisa
  bocorin** baris tenant lain.

```text
iam_rls + app.tenant_id = A → SELECT * FROM projects (tanpa WHERE) → cuma baris tenant A
iam_rls + app.tenant_id kosong → 0 baris tenant (fail-closed)
superuser (jalur tak terbungkus) → semua baris (RLS di-bypass; andalkan WHERE app)
```

---

## 5. OIDC: client → tenant

Tiap OAuth client milik sebuah tenant (organisasi yang dilayani app-nya):

- `ExchangeAuthorizationCode` mengikat sesi ke **tenant si client** (bukan tenant
  pertama user) dan mensyaratkan user jadi anggota aktif — kalau tidak, login
  ditolak. Jadi login lewat OIDC client sebuah org menghasilkan token ter-scope
  ke **org itu**.
- `RegisterClient` menandai client baru dengan tenant aktif si caller.

---

## 6. Endpoint

| Method & path | Permission | Fungsi |
|---|---|---|
| `GET /me/memberships` | (apa pun, terautentikasi) | Tenant yang diikuti caller (sumber switcher) |
| `POST /auth/switch` | (member) | Terbitkan token untuk tenant/project lain |
| `POST /tenants` | `tenant:write` | Buat tenant (pembuat jadi admin-nya) |
| `GET /tenants` | `tenant:read` | List tenant |
| `POST /projects` | `project:write` | Buat project di tenant aktif |
| `GET /projects` | `project:read` | List project tenant aktif |
| `GET /members` | `member:read` | List member tenant aktif |
| `POST /members` | `member:write` | Tambah member via email |
| `DELETE /members/:userId` | `member:write` | Hapus member |
| `GET /users` | `user:read` | Direktori tenant aktif (member ⋈ profile, satu batch) |

Membuat tenant berjalan dalam satu transaksi: buat tenant → daftarkan pembuat →
beri dia role `admin` **ter-scope ke tenant baru** (role platform tidak terbawa
antar tenant).

---

## 7. Console

Admin console (`iam-console`) dapat **tenant switcher** di sidebar (disuplai
`GET /me/memberships`; memilih satu memanggil `POST /auth/switch` lalu mengadopsi
token baru ke sesi NextAuth), plus halaman **Tenants / Projects / Members** dan
direktori **Users** ter-scope tenant — masing-masing dijaga permission terkait.

---

## 8. Catatan operasional

- **Tidak ada env/secret baru**; migrasi (`000010`–`000013`) jalan saat auth
  start. Roll out auth dulu (dia yang menjalankan migrasi) baru user + gateway.
- **Refresh-token rotation grace** (v0.10): refresh token yang baru dirotasi dan
  dipakai lagi dalam jendela grace pendek akan diterbitkan ulang, bukan memicu
  family-wipe deteksi pencurian — ini memperbaiki lockout sesi saat **refresh
  paralel** yang ditembak OIDC client (mis. NextAuth) ketika access token
  kedaluwarsa. Token yang dicabut via logout tetap ditolak keras.
