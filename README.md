# DevOps Demo

This repo is a first-pass scaffold for the spec in `docs/spec.md`.

What is real today:
- a tiny Go HTTP frontend with chat/proxy endpoints
- a working first-pass operator scaffold for `TinyLLMService`
- a real Terraform demo VM on DigitalOcean with cloud-init bootstrap
- GitOps / bootstrap / docs directory structure

What is still scaffold-only:
- GHCR image pipelines for the operator and demo app
- real observability charts/config

## Repo Layout

- `app/` - tiny Go chat frontend
- `operator/` - TinyLLMService API/controller scaffold
- `bootstrap/` - k3s and Argo CD bootstrap scripts
- `gitops/` - Argo CD app-of-apps manifests
- `terraform/` - DigitalOcean demo infra
- `docs/` - spec, runbook, and demo script

## Requirements

- Go 1.22+
- `kubectl`
- `kustomize` (optional, `kubectl apply -k` also works)
- `terraform` (required for the DigitalOcean demo node)
- a Kubernetes cluster if you want to apply manifests

## Terraform Today

The `terraform/envs/demo` stack now creates a real DigitalOcean droplet.

- it provisions a VM
- it creates a firewall
- it injects your SSH public key
- cloud-init installs k3s automatically on first boot
- cloud-init installs `k9s` and wires `/root/.kube/config`
- cloud-init installs `argocd-admin-password` to print the initial login password

Set `DIGITALOCEAN_TOKEN` before running Terraform.
If your public key is not `~/.ssh/id_ed25519.pub`, set `ssh_public_key_path`.

## Reproduce The Current Code

1. Clone the repo.
2. Run the test suite:

```bash
go test ./...
```

3. Start the demo frontend locally:

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
go run ./app/cmd/server --catalog-path=/tmp/services.json --default-service=tiny-llm
```

## Demo Startup Path

If you want the shortest path to a working demo, do this:

1. Export `DIGITALOCEAN_TOKEN`
2. Run `cd terraform/envs/demo && terraform init && terraform apply`
3. SSH to the output `ssh_command` as `root`
4. Verify k3s with the output `k3s_command`
5. Argo CD is installed automatically by cloud-init
6. Run `ssh root@<droplet-ip> 'argocd-admin-password'` and log in as `admin`
7. Apply the GitOps root app after pointing it at your repo fork
8. Apply `gitops/apps/tiny-llm/manifests/sample-cr.yaml`
9. To open the frontend, Argo CD, and Grafana from your laptop, use the helpers:

```bash
./scripts/port-forward-frontend.sh <droplet-ip>
make demo-ui DROPLET_IP=<droplet-ip>
```

Then open `http://localhost:8081`.
Then open `https://localhost:8080`.
Then open `http://localhost:3000`.

## Build The Operator Binary

```bash
go build -o bin/tiny-llm-operator ./operator/cmd/manager
```

That gives you the controller binary for local runs.
The operator image is built by GitHub Actions and pushed to GHCR as `ghcr.io/victorvwier/tiny-llm-operator:latest`.
The demo app image is built the same way as `ghcr.io/victorvwier/tiny-llm-runner:latest`.

## Apply The Kubernetes Scaffolding

If you already have a cluster, you can inspect the manifests now:

```bash
kubectl apply -k operator/config/default
kubectl apply -k gitops/apps
```

Note: the manifests use placeholder image names and repo URLs. Before real cluster deployment, replace these:
- `ghcr.io/victorvwier/tiny-llm-runner:latest`
- `ghcr.io/victorvwier/tiny-llm-operator:latest`
- `https://github.com/your-org/platform-demo.git`

## Intended Full Flow

The spec this repo follows is:

1. Terraform creates a small VM or two.
2. k3s boots on the first VM.
3. Argo CD installs once.
4. Argo CD syncs namespaces, operator, observability, and the demo app.
5. Applying a `TinyLLMService` creates a backend Deployment, Service, and optional Ingress, plus a shared frontend Deployment and catalog ConfigMap.
6. The frontend serves `/health`, `/generate`, `/api/chat`, `/api/services`, `/slow`, `/error`, and `/config`.
7. Grafana/Prometheus/Beyla show traffic and latency.

## Files To Read Next

- `docs/spec.md` - the source spec
- `docs/runbook.md` - short bootstrap checklist
- `docs/demo-script.md` - live demo flow
