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
- Profile: ramp to **50 VUs**, hold, ramp down. Two scenarios:
  - **Read-path** (steady state): `GET /me` (JWT validate + RBAC) + `GET /users`
    (paginated list) — the dominant real-world IAM traffic. 1 min hold.
  - **Mixed**: same, plus `POST /auth/login` (argon2 verify) on 20% of
    iterations — stresses the deliberately-expensive password path. 2 min hold.
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

## Results — mixed (20% argon2 login)

Adding the argon2 login path. **Both stacks fail ~9% identically** — that's the
login requests saturating argon2 on a small cluster (every `/me` + `/users`
check still passed 100%). The failure is a property of **argon2 cost + node
sizing, not the language**.

| Metric | Go | Rust |
|---|---|---|
| Throughput | **759 req/s** | 751 req/s |
| Latency p50 | **8.21 ms** | 9.23 ms |
| Latency p95 | **20.9 ms** | 22.3 ms |
| Latency max | 1.03 s | **331 ms** |
| Error rate (login saturation) | 8.91% | 8.95% |

## Takeaways

- **Within ~3% of each other.** Go edges ahead on read-path throughput and median
  latency; Rust has the **tighter worst-case tail** under the mixed load (max
  331 ms vs Go's 1.03 s — fewer outliers).
- The dominant cost is **argon2** (login) and **single-node infra**, not the
  runtime. For token-validation + read traffic — the bulk of IAM load — both
  hold **sub-25 ms p95 at hundreds of req/s with zero errors**.
- Practical conclusion: at this scale the **architecture** (per-service DB,
  gRPC, event-driven outbox) and **infra sizing** matter far more than Go vs
  Rust. Both are production-grade.

## Reproduce

In-cluster k6 Job (fairest if you have kubectl): see
[`iam-gitops/bench/k6.yaml`](http://gitea.digitalglobalgrowth.com/Digital-Global-Growth/iam-gitops).
Or off-node from a LAN machine against the ingress:

```bash
# resolve the host to the node IP, hit Traefik directly (no Cloudflare)
k6 run -e BASE_URL=http://gateway.iam-go.svc:8080  bench/load.js   # in-cluster
```

## Caveats

Single-node cluster, warm caches, demo data volumes, LAN network. These numbers
show **relative** behavior under identical conditions, not production capacity.
The point is the comparison, not the absolute figures.
