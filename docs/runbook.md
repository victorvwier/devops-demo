# Runbook

## Bootstrap

1. `terraform apply` in `terraform/envs/demo`
2. install k3s using `bootstrap/scripts/install-k3s-server.sh`
3. install Argo CD using `bootstrap/scripts/install-argocd.sh`
4. apply `gitops/root/root-app.yaml`

## Demo flow

1. verify `kubectl get nodes`
2. verify Argo CD apps sync
3. apply `gitops/apps/tiny-llm/manifests/sample-cr.yaml`
4. hit `/health`, `/generate`, `/slow`, `/error`, `/config`
