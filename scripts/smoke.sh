#!/usr/bin/env bash
# End-to-end smoke test for the IAM gateway.
# Usage: ./scripts/smoke.sh [BASE_URL]
# Proves: auth flow, JWT refresh, refresh-token revocation, and granular RBAC.
set -euo pipefail

BASE="${1:-http://localhost:8080}"
ADMIN_EMAIL="${BOOTSTRAP_ADMIN_EMAIL:-admin@iam.local}"
ADMIN_PASS="${BOOTSTRAP_ADMIN_PASSWORD:-admin12345}"
USER_EMAIL="alice+$RANDOM@iam.local"
USER_PASS="alicepass123"

pass=0; fail=0
green() { printf '\033[32m%s\033[0m\n' "$1"; }
red()   { printf '\033[31m%s\033[0m\n' "$1"; }

# json <body> <key> -> prints value of top-level string/number key
json() { python3 -c 'import sys,json;d=json.load(sys.stdin);print(d.get(sys.argv[1],""))' "$2" <<<"$1"; }

# check <label> <expected_status> <actual_status>
check() {
  if [ "$2" = "$3" ]; then green "PASS  $1 (HTTP $3)"; pass=$((pass+1));
  else red "FAIL  $1 (expected $2, got $3)"; fail=$((fail+1)); fi
}

# req METHOD PATH [JSON_BODY] [BEARER]  -> sets RESP (body) and CODE (status)
req() {
  local method="$1" path="$2" body="${3:-}" token="${4:-}"
  local args=(-s -o /tmp/smoke_body -w '%{http_code}' -X "$method" "$BASE$path")
  [ -n "$body" ]  && args+=(-H 'Content-Type: application/json' -d "$body")
  [ -n "$token" ] && args+=(-H "Authorization: Bearer $token")
  CODE=$(curl "${args[@]}")
  RESP=$(cat /tmp/smoke_body)
}

# req_until METHOD PATH BODY TOKEN EXPECTED -> retries until CODE==EXPECTED (or ~6s).
# Profiles are now created by an async UserRegistered event, so reads that
# depend on the profile may need a brief retry.
req_until() {
  local expected="$5"
  for _ in $(seq 1 20); do
    req "$1" "$2" "$3" "$4"
    [ "$CODE" = "$expected" ] && return 0
    sleep 0.3
  done
}

echo "== Smoke test against $BASE =="

# 0) Admin login (bootstrap account)
req POST /auth/login "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASS\"}"
check "admin login" 200 "$CODE"
ADMIN_ACCESS=$(json "$RESP" access_token)

# 1) Register a normal user
req POST /auth/register "{\"email\":\"$USER_EMAIL\",\"password\":\"$USER_PASS\"}"
check "register user" 201 "$CODE"
USER_ID=$(json "$RESP" user_id)

# 2) Login as the normal user
req POST /auth/login "{\"email\":\"$USER_EMAIL\",\"password\":\"$USER_PASS\"}"
check "user login" 200 "$CODE"
USER_ACCESS=$(json "$RESP" access_token)
USER_REFRESH=$(json "$RESP" refresh_token)

# 3) Read own profile
req GET /users/me "" "$USER_ACCESS"
check "get own profile" 200 "$CODE"

# 4) RBAC: normal user lacks user:read -> 403 reading another user
req GET "/users/$USER_ID" "" "$USER_ACCESS"
check "user denied user:read (RBAC)" 403 "$CODE"

# 5) Admin holds user:read -> 200
req GET "/users/$USER_ID" "" "$ADMIN_ACCESS"
check "admin allowed user:read" 200 "$CODE"

# 6) BOLA: a normal user may edit their OWN profile but NOT someone else's
VICTIM_EMAIL="victim+$RANDOM@iam.local"
req POST /auth/register "{\"email\":\"$VICTIM_EMAIL\",\"password\":\"victimpass123\"}"
VICTIM_ID=$(json "$RESP" user_id)
req PATCH "/users/$USER_ID" "{\"bio\":\"updated by self\"}" "$USER_ACCESS"
check "user updates own profile" 200 "$CODE"
req PATCH "/users/$VICTIM_ID" "{\"bio\":\"hacked\"}" "$USER_ACCESS"
check "user denied editing other's profile (BOLA)" 403 "$CODE"

# 7) Refresh rotates the token pair
req POST /auth/refresh "{\"refresh_token\":\"$USER_REFRESH\"}"
check "refresh token" 200 "$CODE"
USER_ACCESS=$(json "$RESP" access_token)
USER_REFRESH2=$(json "$RESP" refresh_token)

# 8) Logout revokes BOTH the refresh token and the access token
req POST /auth/logout "{\"refresh_token\":\"$USER_REFRESH2\"}" "$USER_ACCESS"
check "logout" 200 "$CODE"

# 9) Access token is revoked after logout (jti denylist)
req GET /users/me "" "$USER_ACCESS"
check "access token revoked after logout" 401 "$CODE"

# 10) Revoked refresh token -> 401
req POST /auth/refresh "{\"refresh_token\":\"$USER_REFRESH2\"}"
check "revoked refresh rejected" 401 "$CODE"

# 11) Admin promotes the user to admin
req POST "/users/$USER_ID/roles" "{\"role\":\"admin\"}" "$ADMIN_ACCESS"
check "assign admin role" 200 "$CODE"

# 12) User logs in again -> new token now resolves admin permissions (dynamic RBAC)
req POST /auth/login "{\"email\":\"$USER_EMAIL\",\"password\":\"$USER_PASS\"}"
USER_ACCESS=$(json "$RESP" access_token)
# Victim's profile is created by the async UserRegistered event → may need a retry.
req_until GET "/users/$VICTIM_ID" "" "$USER_ACCESS" 200
check "promoted user now allowed user:read" 200 "$CODE"

# 13) Real delete: identity is removed, the user can no longer log in
req DELETE "/users/$VICTIM_ID" "" "$ADMIN_ACCESS"
check "admin deletes user" 200 "$CODE"
req POST /auth/login "{\"email\":\"$VICTIM_EMAIL\",\"password\":\"victimpass123\"}"
check "deleted user cannot log in" 401 "$CODE"

# ── v0.4 event-driven profile creation ──────────────────────
# A fresh user that never calls /users/me (no lazy heal): the profile must be
# created purely by the async UserRegistered event (outbox → NATS → user svc).
EVT_EMAIL="evt+$RANDOM@iam.local"
req POST /auth/register "{\"email\":\"$EVT_EMAIL\",\"password\":\"evtpass123\"}"
EVT_ID=$(json "$RESP" user_id)
req_until GET "/users/$EVT_ID" "" "$ADMIN_ACCESS" 200
check "profile created via UserRegistered event" 200 "$CODE"

# ── v0.2 Security+ ──────────────────────────────────────────
V2_EMAIL="zoe+$RANDOM@iam.local"
req POST /auth/register "{\"email\":\"$V2_EMAIL\",\"password\":\"zoepass123\"}"
check "v0.2 register" 201 "$CODE"

# 14) Password reset: request -> reset -> login with new password
req POST /auth/password-reset/request "{\"email\":\"$V2_EMAIL\"}"
check "password reset requested" 200 "$CODE"
RESET_TOKEN=$(json "$RESP" dev_token)
req POST /auth/password-reset "{\"token\":\"$RESET_TOKEN\",\"new_password\":\"zoenewpass123\"}"
check "password reset applied" 200 "$CODE"
req POST /auth/login "{\"email\":\"$V2_EMAIL\",\"password\":\"zoenewpass123\"}"
check "login with new password" 200 "$CODE"
req POST /auth/login "{\"email\":\"$V2_EMAIL\",\"password\":\"zoepass123\"}"
check "old password rejected after reset" 401 "$CODE"

# 15) Email verification: request -> verify
req POST /auth/verify-email/request "{\"email\":\"$V2_EMAIL\"}"
check "email verification requested" 200 "$CODE"
VERIFY_TOKEN=$(json "$RESP" dev_token)
req POST /auth/verify-email "{\"token\":\"$VERIFY_TOKEN\"}"
check "email verified" 200 "$CODE"

# 16) Account lockout after repeated failures
LOCK_EMAIL="lock+$RANDOM@iam.local"
req POST /auth/register "{\"email\":\"$LOCK_EMAIL\",\"password\":\"lockpass123\"}"
for _ in 1 2 3 4 5; do req POST /auth/login "{\"email\":\"$LOCK_EMAIL\",\"password\":\"WRONG\"}"; done
req POST /auth/login "{\"email\":\"$LOCK_EMAIL\",\"password\":\"lockpass123\"}"
check "account locked after 5 failures" 401 "$CODE"

# 17) Audit log readable by admin
req GET "/audit?limit=5" "" "$ADMIN_ACCESS"
check "admin reads audit log" 200 "$CODE"

echo "== $pass passed, $fail failed =="
[ "$fail" -eq 0 ]
