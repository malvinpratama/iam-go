// k6 load test for the IAM gateway. Same script runs against either stack.
//
//   k6 run -e BASE_URL=https://iam-go.<domain> -e ADMIN_PASSWORD=ChangeMeAdmin-2026 bench/load.js
//
// Exercises the full path gateway -> auth/user -> Postgres. Per-endpoint timings
// are tagged so you can read p95 per route in the k6 summary.
//
// Multi-tenant overhead (v0.10): a few routes now do extra work —
//   GET /me        : ValidateToken also checks active membership
//   GET /users     : active-tenant directory = ListMembers ⋈ batch GetProfiles
//   GET /members   : read wrapped in a tx as iam_rls + set_config (RLS enforced)
//   GET /projects  : same RLS-wrapped read
//   GET /roles     : plain query (no tx) — the baseline to compare RLS reads against
// Compare /members or /projects (RLS-wrapped, runs a tx + SET LOCAL ROLE) against
// /roles (plain query) to isolate the RLS/tx cost; compare /me + /users against
// v0.9.2 for the cross-version delta.
import http from "k6/http";
import { check, sleep } from "k6";

const BASE = __ENV.BASE_URL || "http://localhost:8080";
const ADMIN = JSON.stringify({
  email: __ENV.ADMIN_EMAIL || "admin@iam.local",
  password: __ENV.ADMIN_PASSWORD || "admin12345",
});
const JSON_HDR = { headers: { "Content-Type": "application/json" } };

export const options = {
  scenarios: {
    load: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: "30s", target: 50 },
        { duration: "2m", target: 50 },
        { duration: "30s", target: 0 },
      ],
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<500"],
    // Per-route budgets so a regression on the RLS-wrapped reads is visible.
    "http_req_duration{route:me}": ["p(95)<400"],
    "http_req_duration{route:members}": ["p(95)<450"],
    "http_req_duration{route:roles}": ["p(95)<400"],
  },
};

export function setup() {
  const res = http.post(`${BASE}/auth/login`, ADMIN, JSON_HDR);
  check(res, { "admin login 200": (r) => r.status === 200 });
  return { token: res.json("access_token") };
}

function get(path, auth, route) {
  return http.get(`${BASE}${path}`, { headers: auth.headers, tags: { route } });
}

export default function (data) {
  const auth = { headers: { Authorization: `Bearer ${data.token}` } };

  check(get("/me", auth, "me"), { "me 200": (r) => r.status === 200 });
  check(get("/users", auth, "users"), { "users 200": (r) => r.status === 200 });

  // Tenant-scoped admin reads: /members + /projects run under RLS (tx as iam_rls
  // + set_config); /roles is the plain-query baseline for the same caller.
  check(get("/members", auth, "members"), { "members 200": (r) => r.status === 200 });
  check(get("/projects", auth, "projects"), { "projects 200": (r) => r.status === 200 });
  check(get("/roles", auth, "roles"), { "roles 200": (r) => r.status === 200 });

  if (Math.random() < 0.2) {
    http.post(`${BASE}/auth/login`, ADMIN, JSON_HDR);
  }
  sleep(0.1);
}
