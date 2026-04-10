# mytasks on k3s

This deployment is configured for your cluster and conventions:

- Namespace: `mytasks`
- Image: `naragon/mytasks:latest`
- Ingress: `todo.sandragon.org`
- Ingress controller: Traefik (`ingressClassName: traefik`, entrypoint `web`)
- Persistence: `local-path` PVC mounted at `/data` for SQLite
- TLS: managed externally by Cloudflare (no in-cluster TLS secret/cert-manager)

## Files

- `namespace.yaml`
- `configmap.yaml`
- `pvc.yaml`
- `deployment.yaml`
- `service.yaml`
- `ingress.yaml`
- `kustomization.yaml`
- `PLAN.md`

## Build and push image (ARM64)

```bash
IMAGE=naragon/mytasks:latest

docker buildx build \
  --platform linux/arm64 \
  -t "$IMAGE" \
  --push .
```

## Deploy

```bash
kubectl apply -k deploy/k3s
kubectl -n mytasks rollout status deployment/mytasks
```

## Verify

```bash
kubectl -n mytasks get pods,svc,ingress,pvc
kubectl -n mytasks logs deploy/mytasks --tail=100
```

## Access

- `http://todo.sandragon.org`

(Cloudflare will handle external TLS/HTTPS according to your existing setup.)

## Why single replica?

`mytasks` uses SQLite. The deployment is intentionally:

- `replicas: 1`
- `strategy: Recreate`

This avoids concurrent-writer issues during rolling updates.
