# Security

🌐 **English** | [Bahasa Indonesia](../id/security.md)

How this IAM is hardened, end to end — from token handling to the cluster it runs
on. The same model is implemented in both the [Go](https://github.com/malvinpratama/iam-go)
and [Rust](https://github.com/malvinpratama/iam-rust) stacks; the deployment lives
in [iam-gitops](https://github.com/malvinpratama/iam-gitops).

The guiding principle throughout is **defense in depth and fail-closed**: every
control has a backstop, and when something is misconfigured the system denies
rather than allows.

---

## Authentication & sessions

- **Password hashing** — Argon2id (memory-hard) with per-user salts. Login runs a
  constant-time compare against a dummy hash for unknown users so response timing
  doesn't leak account existence.
- **JWT access tokens** — short-lived (15 min), RS256-signed; the public keys are
  published as a JWKS. Every request is re-validated server-side (not just by
  signature): the token's `jti` is checked against a **revocation denylist**
  (Redis, shared across replicas, with a Postgres fallback), and the account /
  tenant membership is re-checked.
- **Refresh tokens** — long-lived, rotating, and revocable. Rotation includes
  **reuse detection**: replaying an already-rotated token is rejected, but with a
  short **grace window** so the concurrent refreshes a browser fires at expiry
  don't tear the session down (the Auth0/Okta "reuse interval" pattern).
- **Account lockout** — repeated password (and **MFA**) failures lock the account
  for a cooldown, throttling brute force.
- **Self-service password change** — `POST /auth/password` verifies the current
  password, sets the new one, and revokes every refresh token for the user.

## Two-factor authentication

- Opt-in **TOTP** (RFC 6238) with one-time recovery codes; login becomes a
  challenge (password → short-lived MFA token → TOTP/recovery code → tokens).
- **TOTP secrets are encrypted at rest** with **AES-256-GCM** (key from
  `TOTP_ENC_KEY`). The shared secret can't be hashed — it must be recoverable to
  compute the rolling code — so it's encrypted with a versioned envelope
  (`enc:v1:<nonce‖ciphertext>`), random nonce per write. The scheme is backward
  compatible: a pre-encryption plaintext secret is read transparently and upgraded
  to ciphertext on the next enroll, so enabling it needs no data migration.
- Recovery codes are **hashed** (one-way), never encrypted.

## OIDC / OAuth2 provider

- Authorization Code flow with **PKCE** (S256); authorization codes are
  **single-use** and short-lived.
- `redirect_uri` and `post_logout_redirect_uri` are validated against the client's
  registered allow-list (open-redirect guard).
- Client secrets are stored as SHA-256 hashes; the browser login/consent and the
  TOTP step are rate-limited (no password/OTP oracle).
- ID tokens and access tokens are RS256-signed and carry the tenant binding.

## Authorization & tenant isolation

RBAC is **role → permission**, evaluated per request and **scoped to the token's
tenant (and optional project)** — the same user can hold different permissions in
different organizations. Isolation is enforced at **two independent layers**:

1. **Application layer** — every scoped query filters by the active `tenant_id`
   (and project), and the gateway re-checks the required permission per route while
   each service re-checks again (defense in depth — the gateway is not trusted
   alone).
2. **PostgreSQL Row-Level Security** — a **fail-closed** backstop. Scoped reads and
   writes run inside a transaction as a non-superuser role (`iam_rls`) with the
   tenant set via `SET LOCAL`; RLS policies (including `WITH CHECK` on writes)
   ensure a query that *forgets* its `WHERE`, or a cross-tenant `INSERT`, still
   cannot cross the boundary.

Additional tenant guards: audit events are stamped with and filtered by tenant; a
cross-tenant `GET /users/:id` returns **404** (no existence leak); API keys are
bound to a tenant/project and re-checked on use; **suspending a tenant**
immediately invalidates its members' tokens.

> The runtime currently connects as a superuser that *bypasses* RLS, so RLS only
> bites inside the `iam_rls`-wrapped transactions today. A non-superuser
> connection-role (`iam_app`) is prepared for a future cutover that makes RLS
> enforce everywhere — see [Roadmap](#known-gaps--roadmap).

## Secrets

- **No plaintext secrets in git.** Kubernetes secrets are committed as **Sealed
  Secrets** (encrypted with the in-cluster controller's key); only the controller
  can decrypt them into real `Secret`s.
- **Placeholder rejection** — a production process (`APP_ENV=production`) refuses
  to boot if `JWT_SECRET`, `BOOTSTRAP_ADMIN_PASSWORD`, `INTERNAL_SERVICE_TOKEN`, or
  the database password still carry a known demo/placeholder marker. Insecure
  defaults can't reach production silently.

## Service-to-service

- The gateway authenticates to the auth/user services with a shared
  **`INTERNAL_SERVICE_TOKEN`**, enforced **fail-closed**: a missing/empty token
  rejects every internal call (an explicit `INTERNAL_AUTH_OPTIONAL=true` is
  required to relax it for local dev). The gateway never forwards client-supplied
  identity headers — it rebuilds them from the validated token, so they can't be
  spoofed.
- *Planned:* mTLS between the gateway and services (cert generation already ships;
  wiring is pending).

## Network

- **Default-deny NetworkPolicies**: every pod denies all ingress except an explicit
  allow-list — the gateway only from the ingress controller, auth/user only from
  the gateway, each database only from its own service, NATS/Redis only from their
  callers. A compromised pod can't pivot laterally.

## Container & pod hardening

Every application container runs:

- **non-root** with a fixed UID and `runAsNonRoot`,
- a **read-only root filesystem** (a small `emptyDir` at `/tmp` for scratch),
- **all Linux capabilities dropped**, no privilege escalation, `seccompProfile:
  RuntimeDefault`,
- **CPU/memory requests + limits**, and
- the **service-account token unmounted** (`automountServiceAccountToken: false`).

## Supply chain & deployment

- **Pure GitOps** — the cluster state lives in git ([iam-gitops](https://github.com/malvinpratama/iam-gitops));
  ArgoCD reconciles it. No imperative changes survive.
- **Immutable image pinning** — deployments reference `sha-<commit>` image tags,
  never `:latest`. Every rollout is deterministic, auditable, and reversible
  (roll back by pointing the tag at a previous commit).

## Observability

Distributed tracing (OpenTelemetry → Jaeger), Prometheus metrics, and an
`X-Request-Id` correlated across logs and traces — so an auth failure or a denied
request can be followed end to end. Sensitive material (e.g. email-verification /
password-reset token bodies) is suppressed from logs outside development.

## Known gaps & roadmap

Honest about what isn't done yet:

- **mTLS** gateway↔services — deferred (plaintext gRPC on the internal network
  today, mitigated by NetworkPolicies + the internal token).
- **Egress NetworkPolicies** — only ingress is default-denied so far; egress is
  open (DNS, DB, NATS, Redis, OIDC discovery).
- **Least-privilege DB cutover** — the `iam_app` non-superuser role is prepared but
  the connection still runs as the RLS-bypassing superuser; the cutover needs the
  remaining hot paths wrapped first.
- **OAuth tables under RLS** — `oauth_authorization_codes` / `oauth_consents` are
  not yet tenant-scoped at the database layer.

---

*Found something? Security issues are welcome via a private report — see
[CONTRIBUTING.md](../../CONTRIBUTING.md).*
