// k6 load test for the IAM gateway. Same script runs against either stack.
//
//   k6 run -e BASE_URL=https://iam-go.<domain> -e ADMIN_PASSWORD=ChangeMeAdmin-2026 bench/load.js
//
// Exercises the full path gateway -> auth/user -> Postgres:
//   GET /me           (ValidateToken + roles/permissions)
//   GET /users        (paginated profile list)
//   POST /auth/login  (argon2 verify + token issue), 20% of iterations
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
  },
};

export function setup() {
  const res = http.post(`${BASE}/auth/login`, ADMIN, JSON_HDR);
  check(res, { "admin login 200": (r) => r.status === 200 });
  return { token: res.json("access_token") };
}

export default function (data) {
  const auth = { headers: { Authorization: `Bearer ${data.token}` } };
  check(http.get(`${BASE}/me`, auth), { "me 200": (r) => r.status === 200 });
  check(http.get(`${BASE}/users?page=1&page_size=20`, auth), {
    "users 200": (r) => r.status === 200,
  });
  if (Math.random() < 0.2) {
    http.post(`${BASE}/auth/login`, ADMIN, JSON_HDR);
  }
  sleep(0.1);
}
