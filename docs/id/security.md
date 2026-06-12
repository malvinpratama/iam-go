# Security

🌐 [English](../en/security.md) | **Bahasa Indonesia**

Bagaimana IAM ini di-harden, end to end — dari penanganan token sampai cluster
tempat ia berjalan. Model yang sama diimplementasikan di kedua stack
[Go](https://github.com/malvinpratama/iam-go) dan
[Rust](https://github.com/malvinpratama/iam-rust); deployment-nya ada di
[iam-gitops](https://github.com/malvinpratama/iam-gitops).

Prinsip pemandu di sepanjang dokumen ini adalah **defense in depth dan
fail-closed**: setiap kontrol punya pengaman cadangan, dan ketika ada yang salah
konfigurasi sistem memilih menolak ketimbang mengizinkan.

---

## Authentication & sessions

- **Password hashing** — Argon2id (memory-hard) dengan salt per-user. Login
  menjalankan perbandingan constant-time terhadap hash dummy untuk user yang tak
  dikenal, sehingga timing respons tidak membocorkan keberadaan akun.
- **JWT access token** — berumur pendek (15 menit), ditandatangani RS256; public
  key dipublikasikan sebagai JWKS. Setiap request divalidasi ulang di sisi server
  (bukan sekadar lewat signature): `jti` token dicek terhadap **denylist
  revocation** (Redis, dibagi lintas replika, dengan fallback Postgres), dan
  akun / membership tenant dicek ulang.
- **Refresh token** — berumur panjang, berotasi, dan bisa di-revoke. Rotasi
  mencakup **deteksi reuse**: memutar ulang token yang sudah dirotasi akan
  ditolak, tetapi dengan **jendela grace** pendek sehingga refresh konkuren yang
  ditembak browser saat expiry tidak meruntuhkan sesi (pola "reuse interval" ala
  Auth0/Okta).
- **Account lockout** — kegagalan password (dan **MFA**) berulang mengunci akun
  selama cooldown, menghambat brute force.
- **Ubah password swalayan** — `POST /auth/password` memverifikasi password saat
  ini, menyetel yang baru, dan mencabut setiap refresh token milik user.

## Two-factor authentication

- **TOTP** opt-in (RFC 6238) dengan recovery code sekali pakai; login menjadi
  sebuah challenge (password → MFA token berumur pendek → kode TOTP/recovery →
  token).
- **Secret TOTP dienkripsi at rest** dengan **AES-256-GCM** (kunci dari
  `TOTP_ENC_KEY`). Secret bersama tidak bisa di-hash — ia harus dapat dipulihkan
  untuk menghitung rolling code — jadi dienkripsi dengan envelope ber-versi
  (`enc:v1:<nonce‖ciphertext>`), nonce acak per write. Skema ini backward
  compatible: secret plaintext pra-enkripsi dibaca secara transparan dan
  di-upgrade ke ciphertext pada enroll berikutnya, sehingga mengaktifkannya tak
  butuh migrasi data.
- Recovery code di-**hash** (one-way), tidak pernah dienkripsi.

## OIDC / OAuth2 provider

- Flow Authorization Code dengan **PKCE** (S256); authorization code bersifat
  **sekali pakai** dan berumur pendek.
- `redirect_uri` dan `post_logout_redirect_uri` divalidasi terhadap allow-list
  terdaftar milik client (penjaga open-redirect).
- Client secret disimpan sebagai hash SHA-256; login/consent di browser dan
  langkah TOTP di-rate-limit (tidak ada oracle password/OTP).
- ID token dan access token ditandatangani RS256 dan membawa binding tenant.

## Authorization & tenant isolation

RBAC bersifat **role → permission**, dievaluasi per request dan **ter-scope ke
tenant token (dan project opsional)** — user yang sama bisa memegang permission
berbeda di organisasi berbeda. Isolasi ditegakkan di **dua lapis independen**:

1. **Application layer** — setiap query ter-scope memfilter berdasarkan
   `tenant_id` aktif (dan project), dan gateway mengecek ulang permission yang
   dibutuhkan per route sementara tiap service mengecek ulang lagi (defense in
   depth — gateway tidak dipercaya sendirian).
2. **PostgreSQL Row-Level Security** — pengaman cadangan **fail-closed**. Read
   dan write ter-scope berjalan di dalam transaksi sebagai role non-superuser
   (`iam_rls`) dengan tenant disetel via `SET LOCAL`; policy RLS (termasuk
   `WITH CHECK` pada write) memastikan query yang *lupa* `WHERE`-nya, atau
   `INSERT` lintas-tenant, tetap tak bisa melintasi batas.

Penjaga tenant tambahan: event audit dicap dan difilter berdasarkan tenant;
`GET /users/:id` lintas-tenant mengembalikan **404** (tidak membocorkan
keberadaan); API key terikat ke tenant/project dan dicek ulang saat dipakai;
**men-suspend sebuah tenant** seketika meng-invalidasi token para member-nya.

> Runtime saat ini connect sebagai superuser yang *mem-bypass* RLS, jadi RLS
> hanya menggigit di dalam transaksi yang dibungkus `iam_rls` hari ini. Sebuah
> connection-role non-superuser (`iam_app`) sudah disiapkan untuk cutover di masa
> depan yang membuat RLS menegakkan di mana-mana — lihat
> [Roadmap](#known-gaps--roadmap).

## Secrets

- **Tidak ada secret plaintext di git.** Secret Kubernetes di-commit sebagai
  **Sealed Secrets** (dienkripsi dengan kunci controller dalam cluster); hanya
  controller yang bisa men-dekripsi-nya menjadi `Secret` sungguhan.
- **Penolakan placeholder** — proses produksi (`APP_ENV=production`) menolak boot
  jika `JWT_SECRET`, `BOOTSTRAP_ADMIN_PASSWORD`, `INTERNAL_SERVICE_TOKEN`, atau
  password database masih membawa penanda demo/placeholder yang dikenal. Default
  yang tidak aman tak bisa mencapai produksi secara diam-diam.

## Service-to-service

- Gateway mengautentikasi ke service auth/user dengan **`INTERNAL_SERVICE_TOKEN`**
  bersama, ditegakkan **fail-closed**: token yang hilang/kosong menolak setiap
  panggilan internal (`INTERNAL_AUTH_OPTIONAL=true` eksplisit diperlukan untuk
  melonggarkannya bagi dev lokal). Gateway tak pernah meneruskan header identitas
  yang dipasok client — ia membangunnya ulang dari token yang sudah divalidasi,
  jadi tak bisa di-spoof.
- *Direncanakan:* mTLS antara gateway dan service (pembuatan sertifikat sudah
  ada; wiring masih tertunda).

## Network

- **NetworkPolicy default-deny**: setiap pod menolak seluruh ingress kecuali
  allow-list eksplisit — gateway hanya dari ingress controller, auth/user hanya
  dari gateway, tiap database hanya dari service-nya sendiri, NATS/Redis hanya
  dari pemanggilnya. Pod yang ter-kompromi tak bisa pivot secara lateral.

## Container & pod hardening

Setiap container aplikasi berjalan:

- **non-root** dengan UID tetap dan `runAsNonRoot`,
- **root filesystem read-only** (sebuah `emptyDir` kecil di `/tmp` untuk scratch),
- **semua Linux capability di-drop**, tanpa privilege escalation, `seccompProfile:
  RuntimeDefault`,
- **request + limit CPU/memory**, dan
- **token service-account tidak di-mount** (`automountServiceAccountToken: false`).

## Supply chain & deployment

- **Pure GitOps** — state cluster ada di git
  ([iam-gitops](https://github.com/malvinpratama/iam-gitops)); ArgoCD
  merekonsiliasinya. Tidak ada perubahan imperatif yang bertahan.
- **Pinning image immutable** — deployment mereferensikan tag image
  `sha-<commit>`, tak pernah `:latest`. Setiap rollout deterministik, auditable,
  dan reversible (roll back dengan mengarahkan tag ke commit sebelumnya).

## Observability

Distributed tracing (OpenTelemetry → Jaeger), metrik Prometheus, dan
`X-Request-Id` yang dikorelasikan lintas log dan trace — sehingga kegagalan auth
atau request yang ditolak bisa ditelusuri end to end. Material sensitif (mis. body
token verifikasi-email / reset-password) ditekan dari log di luar development.

## Known gaps & roadmap

Jujur soal yang belum selesai:

- **mTLS** gateway↔service — ditunda (gRPC plaintext di jaringan internal hari
  ini, dimitigasi oleh NetworkPolicy + token internal).
- **Egress NetworkPolicy** — baru ingress yang default-denied sejauh ini; egress
  masih terbuka (DNS, DB, NATS, Redis, OIDC discovery).
- **Cutover DB least-privilege** — role non-superuser `iam_app` sudah disiapkan
  tapi koneksi masih berjalan sebagai superuser yang mem-bypass RLS; cutover
  butuh sisa hot path dibungkus dulu.
- **Tabel OAuth di bawah RLS** — `oauth_authorization_codes` / `oauth_consents`
  belum ter-scope tenant di lapisan database.

---

*Menemukan sesuatu? Isu keamanan diterima via laporan privat — lihat
[CONTRIBUTING.md](../../CONTRIBUTING.md).*
