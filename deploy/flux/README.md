# Flux GitOps setup for mytasks (Option B)

This setup enables **pull-based deployment** from your local k3s cluster:

1. GitHub Actions builds/pushes `naragon/mytasks` image tags on each commit to `main`.
2. Flux running in-cluster watches this repo.
3. Flux image automation updates `deploy/k3s/deployment.yaml` image tag.
4. Flux applies the updated manifest to the cluster.

No inbound access from GitHub to your local network is required.

## Repository layout

- `clusters/rotring/kustomization.yaml` - cluster entrypoint for Flux
- `clusters/rotring/mytasks-image-automation.yaml` - image scan/policy/automation
- `deploy/k3s/deployment.yaml` - image field contains Flux setter annotation

## Prerequisites

- Flux CLI installed locally
- GitHub PAT with repo write/admin scope
- `kubectl` context set to your k3s cluster

## 1) Bootstrap Flux

```bash
export GITHUB_TOKEN=<your-github-token>

flux bootstrap github \
  --token-auth \
  --owner=naragon \
  --repository=mytasks \
  --branch=main \
  --path=clusters/rotring \
  --personal
```

This installs Flux controllers into `flux-system` and configures sync from `clusters/rotring`.

## 2) Verify Flux and image automation

```bash
kubectl -n flux-system get pods
kubectl -n flux-system get gitrepositories,kustomizations
kubectl -n flux-system get imagerepositories,imagepolicies,imageupdateautomations
```

## 3) Observe reconciliation

```bash
flux get sources git -A
flux get kustomizations -A
flux get image repository mytasks -n flux-system
flux get image policy mytasks -n flux-system
flux get image update mytasks -n flux-system
```

## Tag strategy used by CI

GitHub Actions publishes:
- `latest`
- `main-YYYYMMDDHHmmss` (used by Flux policy)
- `sha-<gitsha>`

Flux policy selects the highest numeric timestamp from tags matching `main-...`.
