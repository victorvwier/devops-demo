# Runbook

## Bootstrap

1. export `DIGITALOCEAN_TOKEN`
2. `terraform init` in `terraform/envs/demo`
3. `terraform apply` in `terraform/envs/demo`
4. if needed, override `ssh_public_key_path` to match your local public key
5. SSH to the droplet using the printed command as `root`
6. verify k3s with `k3s kubectl get nodes`
7. Argo CD is installed automatically by cloud-init
8. run `ssh root@<droplet-ip> 'argocd-admin-password'`
9. if you forked the repo, update `gitops/root/root-app.yaml` before `terraform apply`
10. `k9s` is installed and uses `/root/.kube/config`

## Demo flow

1. verify `k3s kubectl get nodes`
2. verify Argo CD apps sync
3. wait for Argo CD to auto-sync the root app and TinyLLM manifests
4. hit `/health`, `/generate`, `/slow`, `/error`, `/config`
5. open the frontend, Argo CD, and Grafana in your laptop browser:

```bash
make demo-ui DROPLET_IP=<droplet-ip>
```

The frontend is available at `http://localhost:8081`.
Argo CD is available at `http://localhost:8080`.
Grafana is available at `http://localhost:3000`.

## If You Are New To Terraform

Terraform is not the thing to start the app itself.

- `init` prepares Terraform to run.
- `apply` makes Terraform compare the config to state and then change real resources.
- in this repo, Terraform now creates a real DigitalOcean VM and firewall.
- cloud-init brings up k3s automatically on first boot.
- cloud-init also installs Argo CD automatically.
