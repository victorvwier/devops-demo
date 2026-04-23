# DevOps Demo

This repo is a first-pass scaffold for the spec in `docs/spec.md`.

What is real today:
- a tiny Go HTTP app with the required endpoints
- a working first-pass operator scaffold for `TinyLLMService`
- GitOps / bootstrap / Terraform / docs directory structure

What is still scaffold-only:
- real cloud Terraform resources
- a fully installable operator image pipeline
- real observability charts/config

## Repo Layout

- `app/` - tiny Go service
- `operator/` - TinyLLMService API/controller scaffold
- `bootstrap/` - k3s and Argo CD bootstrap scripts
- `gitops/` - Argo CD app-of-apps manifests
- `terraform/` - infra placeholders
- `docs/` - spec, runbook, and demo script

## Requirements

- Go 1.22+
- `kubectl`
- `kustomize` (optional, `kubectl apply -k` also works)
- `terraform` (optional, only for the placeholders here)
- a Kubernetes cluster if you want to apply manifests

## Reproduce The Current Code

1. Clone the repo.
2. Run the test suite:

```bash
go test ./...
```

3. Start the demo app locally:

```bash
go run ./app/cmd/server
```

4. In a second terminal, hit the endpoints:

```bash
curl http://localhost:8080/health
curl -X POST http://localhost:8080/generate -d '{"prompt":"hello"}'
curl http://localhost:8080/slow
curl http://localhost:8080/error
curl http://localhost:8080/config
```

5. Change config with flags if needed:

```bash
go run ./app/cmd/server --model-mode=mock --prompt-prefix='Demo:'
```

## Build The Operator Binary

```bash
go build -o bin/tiny-llm-operator ./operator/cmd/manager
```

That gives you the controller binary, but the repo does not yet include a full image build/push flow.

## Apply The Kubernetes Scaffolding

If you already have a cluster, you can inspect the manifests now:

```bash
kubectl apply -k operator/config/default
kubectl apply -k gitops/apps
```

Note: the manifests use placeholder image names and repo URLs. Before real cluster deployment, replace these:
- `ghcr.io/your-org/tiny-llm-runner:latest`
- `ghcr.io/your-org/tiny-llm-operator:latest`
- `https://github.com/your-org/platform-demo.git`

## Intended Full Flow

The spec this repo follows is:

1. Terraform creates a small VM or two.
2. k3s boots on the first VM.
3. Argo CD installs once.
4. Argo CD syncs namespaces, operator, observability, and the demo app.
5. Applying a `TinyLLMService` creates Deployment, Service, ConfigMap, and optional Ingress.
6. The app serves `/health`, `/generate`, `/slow`, `/error`, and `/config`.
7. Grafana/Prometheus/Beyla show traffic and latency.

## Files To Read Next

- `docs/spec.md` - the source spec
- `docs/runbook.md` - short bootstrap checklist
- `docs/demo-script.md` - live demo flow
