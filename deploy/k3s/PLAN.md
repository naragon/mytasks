# Deployment Plan (k3s)

## Scope
Deploy `mytasks` to the k3s cluster using:
- namespace: `mytasks`
- ingress host: `todo.sandragon.org`
- ingress style aligned with `hello-world-deployment` (Traefik `web` entrypoint)
- TLS terminated externally at Cloudflare (no in-cluster TLS config)

## Plan

1. **Container image**
   - Use Docker Hub org pattern from `hello-world-deployment`.
   - Target image: `naragon/mytasks:latest`.
   - Build/push as linux/arm64 for Raspberry Pi worker nodes.

2. **Kubernetes resources**
   - Create namespace `mytasks`.
   - Create app config via ConfigMap (`PORT`, `DB_PATH`).
   - Create PVC (`local-path`, 1Gi) for SQLite persistence.
   - Deploy app (`replicas: 1`, `strategy: Recreate`) for SQLite safety.
   - Expose app via ClusterIP service on port 80 -> container 8080.
   - Create Ingress for `todo.sandragon.org` with Traefik `web` entrypoint.

3. **Validation**
   - Client-side manifest validation via `kubectl apply --dry-run=client -k deploy/k3s`.
   - Runtime checks after apply:
     - rollout status
     - pod logs
     - service/ingress/pvc health

4. **Operational notes**
   - Keep single replica unless DB is moved away from SQLite.
   - Cloudflare handles external TLS; ingress remains HTTP in-cluster.
