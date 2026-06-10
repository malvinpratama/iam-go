# Benchmarks — Go vs Rust

Both stacks implement the **same API** over the **same Postgres/NATS topology**,
so running identical load against each (on the same cluster, identical CPU/memory
requests) is a fair head-to-head.

## Method

- Tool: **[k6](https://k6.io)**, script [`bench/load.js`](bench/load.js).
- Workload per iteration: `GET /me` (token validation + RBAC), `GET /users`
  (paginated list), and `POST /auth/login` (argon2 verify) on 20% of iterations.
  This exercises gateway → auth/user → Postgres end-to-end.
- Profile: ramp to **50 VUs**, hold **2 min**, ramp down.
- Run against the live ingress of each stack:

```bash
k6 run -e BASE_URL=https://iam-go.<domain>   -e ADMIN_PASSWORD=ChangeMeAdmin-2026 bench/load.js
k6 run -e BASE_URL=https://iam-rust.<domain> -e ADMIN_PASSWORD=ChangeMeAdmin-2026 bench/load.js
```

## Results

> _Pending the live k3s deploy. Numbers below are filled from a real run against
> both stacks on the same cluster._

| Metric | Go | Rust |
|---|---|---|
| Requests/s (throughput) | — | — |
| Latency p50 | — | — |
| Latency p95 | — | — |
| Latency p99 | — | — |
| Error rate | — | — |
| Peak RSS (gateway/auth/user) | — | — |

## Caveats

Single-node cluster, warm caches, in-cluster network, demo data volumes. These
numbers indicate **relative** behavior of the two stacks under identical
conditions, not production capacity. The point is the comparison, not the
absolute figures.
