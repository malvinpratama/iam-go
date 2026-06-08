.PHONY: up down logs smoke k8s

## Run the full stack via docker-compose (pulls service images from GHCR).
## Override the tag with IMAGE_TAG (e.g. `make up IMAGE_TAG=dev`).
up:
	cd deploy && [ -f .env ] || cp .env.example .env
	cd deploy && docker compose up -d

down:
	cd deploy && docker compose down -v

logs:
	cd deploy && docker compose logs -f

## End-to-end smoke test against the running gateway
smoke:
	./scripts/smoke.sh http://localhost:8080

## Render the Kubernetes manifests
k8s:
	kubectl kustomize deploy/k8s
