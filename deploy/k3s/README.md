# mytasks on k3s

This deployment uses a shared Kustomize `base` with one overlay and namespace per instance:

- Instances:
  - `mytasks`: namespace `mytasks`, host `todo.sandragon.org`
  - `worktasks`: namespace `worktasks`, host `worktasks.sandragon.org`
- Image: `naragon/mytasks` (auto-updated in `base/deployment.yaml` via Flux Image Automation)
- Ingress controller: Traefik (`ingressClassName: traefik`, entrypoint `web`)
- Persistence: isolated `local-path` PVC (`mytasks-data`) per namespace mounted at `/data` for SQLite
- TLS: managed externally by Cloudflare (no in-cluster TLS secret/cert-manager)

## Layout

- `base/` - shared manifests (`configmap.yaml`, `pvc.yaml`, `deployment.yaml`, `service.yaml`, `ingress.yaml`, `kustomization.yaml`)
- `overlays/mytasks/` - primary instance (`mytasks` namespace, `todo.sandragon.org`)
- `overlays/worktasks/` - second instance (`worktasks` namespace, `worktasks.sandragon.org`)
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
kubectl apply -k deploy/k3s/overlays/mytasks
kubectl -n mytasks rollout status deployment/mytasks

kubectl apply -k deploy/k3s/overlays/worktasks
kubectl -n worktasks rollout status deployment/mytasks
```

## Verify

```bash
kubectl -n mytasks get pods,svc,ingress,pvc
kubectl -n worktasks get pods,svc,ingress,pvc
kubectl -n mytasks logs deploy/mytasks --tail=100
kubectl -n worktasks logs deploy/mytasks --tail=100
```

## Access

- `http://todo.sandragon.org`
- `http://worktasks.sandragon.org`

(Cloudflare will handle external TLS/HTTPS according to your existing setup.)

## Why single replica?

`mytasks` uses SQLite. The deployment is intentionally:

- `replicas: 1`
- `strategy: Recreate`

This avoids concurrent-writer issues during rolling updates.
