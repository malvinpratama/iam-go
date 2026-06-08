# Koleksi API (Postman & Bruno) — iam-go

🌐 [English](../en/api-collections.md) | **Bahasa Indonesia** · [↑ Indeks dokumentasi](README.md)

Dua koleksi **native** berada di root repositori (dibagikan oleh kedua stack —
REST API-nya identik):

- **Postman**: `iam.postman_collection.json` + `iam.postman_environment.json`
- **Bruno**: `IAM — User & Auth (Go - Rust)/` (folder open-collection)

> ⚠️ **JANGAN impor JSON Postman ke Bruno.** Postman menggunakan API `pm.*`
> sedangkan Bruno menggunakan `bru.*`/`res.*`; importer Bruno hanya menerjemahkan
> sebagian skrip dan meninggalkan `pm.collectionVariables.set(...)`, yang gagal saat
> runtime dengan `ReferenceError: pm is not defined`. Gunakan koleksi Bruno native
> sebagai gantinya (Open Collection).

## Postman / Newman

1. Impor kedua file ke Postman.
2. Pilih environment **IAM — Local** (opsional; koleksi juga membawa variabelnya
   sendiri).
3. Jalankan **Auth → Login (admin)**, lalu **Register**, **Login (user)**.
   Post-script Login/Register/Refresh otomatis menyimpan `access_token`, `refresh_token`,
   `token_type`, `admin_access_token`, `user_id` ke dalam **collection variables**
   (`pm.collectionVariables.set`), sehingga request Users/RBAC tidak perlu diedit.

CLI (Newman):
```bash
npx newman run iam.postman_collection.json -e iam.postman_environment.json
```

## Bruno

1. **Open Collection** → pilih folder `IAM — User & Auth (Go - Rust)`.
2. Pilih environment **IAM — Local** (kanan atas).
3. Urutan yang sama: Login (admin) → Register → Login (user) → eksplorasi.
   Post-script menggunakan `bru.setVar(...)` sehingga `{{access_token}}` ter-resolve
   pada request berikutnya — tidak perlu melihat editor environment (nilai yang
   diset skrip bersifat runtime; lihat melalui ikon variabel 👁, bukan editor).

CLI (Bruno):
```bash
cd "IAM — User & Auth (Go - Rust)"
npx @usebruno/cli run --env "IAM — Local" -r
```

## Folder / request

- **Auth**: Register, Login (user), Login (admin), Refresh, Logout
- **Users**: My Identity, Get My Profile, Get User by ID, List Users, Update Profile, Delete User
- **RBAC**: List Roles, List Permissions, Create/Update/Delete Role, Grant/Revoke Permission to Role, Assign/Revoke Role to User
- **Health**: Healthz

Request khusus admin menggunakan `{{admin_access_token}}`; request lingkup pengguna
menggunakan `{{access_token}}`. Untuk melihat penolakan RBAC, panggil **Get User by ID**
dengan token pengguna → `403`.

Lihat [Referensi API](api-reference.md) untuk detail lengkap request/response.

> ⚠️ **Logout kini mencabut access token** (denylist jti), jadi jalankan **Logout paling akhir** — pada run koleksi otomatis penuh, ia membuat request terautentikasi berikutnya jadi tidak valid.
