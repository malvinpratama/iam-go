# Benchmarks — Go vs Rust

Both stacks implement the **same API** over the **same Postgres/NATS topology**,
deployed to the **same single-node k3s cluster** with identical CPU/memory
requests — so running the same load against each is a fair head-to-head.

## Method

- Tool: **[k6](https://k6.io)** (v2.0.0), script [`bench/load.js`](bench/load.js).
- **Off-node load generator**: k6 ran on a separate machine on the **same LAN**,
  hitting the Traefik ingress directly (`Host` override → node IP, **no
  Cloudflare/tunnel**). This keeps the load generator from stealing CPU from the
  system under test and removes edge/tunnel overhead — pure stack comparison.
- Profile: ramp to **50 VUs**, hold, ramp down. Scenarios:
  - **Read-path** (steady state): `GET /me` (JWT validate + RBAC) + `GET /users`
    (paginated list) — the dominant real-world IAM traffic. 1 min hold.
  - **Mixed**: same, plus `POST /auth/login` on 20% of iterations. 2 min hold.
  - **Auth stress**: `POST /auth/login` only, to characterise the rate limiter.
- Run on **2026-06-10**, single-node k3s, warm caches.

## Results — read-path (steady state, 0% errors)

The realistic IAM workload: token validation + listing. Both stacks served it
flawlessly (0 failures over ~70k requests each).

| Metric | Go | Rust |
|---|---|---|
| Throughput | **649 req/s** | 631 req/s |
| Latency p50 | **7.97 ms** | 9.16 ms |
| Latency p90 | **14.3 ms** | 17.1 ms |
| Latency p95 | **17.5 ms** | 21.6 ms |
| Latency max | **132 ms** | 286 ms |
| Error rate | 0.00% | 0.00% |

## Results — mixed (20% login, rate limiter ON)

| Metric | Go | Rust |
|---|---|---|
| Throughput | **759 req/s** | 751 req/s |
| Latency p50 | **8.21 ms** | 9.23 ms |
| Latency p95 | **20.9 ms** | 22.3 ms |
| Latency max | 1.03 s | **331 ms** |
| Error rate | 8.91% | 8.95% |

The ~9% error here is **not** a stack failure — it's the gateway's **per-IP auth
rate limiter** (fixed window, **60 req/min**, guarding `/auth/*`) returning
**HTTP 429** once the test's single IP exceeds 60 logins/min. Every `/me` +
`/users` check still passed 100%. Identical in both stacks → it's a security
control working as designed, not saturation.

## Results — mixed (rate limiter OFF — true argon2 cost)

Re-run with `AUTH_RATE_LIMIT=0` so every login actually runs **argon2** (no 429).
The real cost of password hashing surfaces — and the ranking **flips**:

| Metric | Go | Rust |
|---|---|---|
| Throughput | 375 req/s | **400 req/s** |
| Latency p50 | 44.8 ms | **43.9 ms** |
| Latency p95 | 175 ms | **146 ms** |
| Latency max | **2.96 s** | 5.9 s |
| Error rate | 0.00% | 0.00% |

**0 errors** over ~64k/68k requests — argon2 does **not** saturate or time out at
50 VUs, it just adds latency (p50 jumps 8→45 ms vs read-path). This is the final
proof the earlier ~9% was purely the rate limiter. On this **CPU-bound** path
**Rust leads** (higher throughput, lower p95) — the mirror image of the read path.

## Results — auth stress (login-only, status breakdown)

Hammering `POST /auth/login` from one IP confirms the limiter (429 is rejected
cheaply, before argon2 — hence the very high reject throughput):

| | Go | Rust |
|---|---|---|
| Logins allowed (2xx) | 60 | 60 |
| Rate-limited (429) | 226,616 | 228,549 |
| 5xx | **0** | 2,391 (~1%) |

Both admit exactly **60 logins/min/IP** then 429 the rest — brute-force
protection. One real difference: under this extreme ~5k req/s reject load, Rust
emitted ~1% 5xx while Go emitted none.

## Takeaways

- **The workload picks the winner — and it's close either way:**
  - **Read/validate path (I/O-bound):** Go edges ahead — 649 vs 631 req/s,
    p95 17.5 vs 21.6 ms.
  - **Login/argon2 path (CPU-bound):** Rust edges ahead — 400 vs 375 req/s,
    p95 146 vs 175 ms.
  Near mirror-image results: **Go for I/O-bound reads, Rust for CPU-bound hashing.**
- **The earlier ~9% error was the auth rate limiter (429), by design** — not
  argon2, not the runtime. With the limiter off the same mixed load runs at
  **0% errors**; argon2 adds latency but never saturates at 50 VUs.
- At this scale the **architecture** (per-service DB, gRPC, event-driven outbox)
  and **security controls** matter far more than Go vs Rust. Both are
  production-grade.

## Reproduce

In-cluster k6 Job (fairest if you have kubectl): see
[`iam-gitops/bench/k6.yaml`](http://gitea.digitalglobalgrowth.com/Digital-Global-Growth/iam-gitops).
Or off-node from a LAN machine against the ingress. Note the **60 req/min/IP**
auth limit — raise it (or spread source IPs) to load-test the login path itself.

## Caveats

Single-node cluster, warm caches, demo data volumes, LAN network. These numbers
show **relative** behavior under identical conditions, not production capacity.
The point is the comparison, not the absolute figures.
