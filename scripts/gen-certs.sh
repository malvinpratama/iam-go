#!/usr/bin/env bash
# Generate self-signed certificates for local TLS/mTLS experiments.
# Optional: TLS is OFF by default. These certs let you try TLS locally;
# in production use a real CA / cert-manager. Outputs to deploy/tls/.
set -euo pipefail

OUT="$(cd "$(dirname "$0")/.." && pwd)/deploy/tls"
mkdir -p "$OUT"
cd "$OUT"

# Root CA
openssl req -x509 -newkey rsa:4096 -nodes -keyout ca.key -out ca.crt -days 3650 \
  -subj "/CN=iam-local-ca" 2>/dev/null

# Per-service certs (gateway, auth, user) signed by the CA, with SAN = service name
for svc in gateway auth user; do
  openssl req -newkey rsa:2048 -nodes -keyout "$svc.key" -out "$svc.csr" \
    -subj "/CN=$svc" 2>/dev/null
  openssl x509 -req -in "$svc.csr" -CA ca.crt -CAkey ca.key -CAcreateserial \
    -out "$svc.crt" -days 825 \
    -extfile <(printf "subjectAltName=DNS:%s,DNS:localhost" "$svc") 2>/dev/null
  rm -f "$svc.csr"
done

echo "Certificates written to $OUT"
echo "Enable TLS by setting TLS_ENABLED=true and the *_CERT/*_KEY/CA_CERT paths."
