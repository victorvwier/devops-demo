terraform {
  required_version = ">= 1.5.0"
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
  }
}

resource "digitalocean_tag" "this" {
  name = var.name
}

locals {
  k3s_server_script      = file(abspath("${path.module}/../../../bootstrap/scripts/install-k3s-server.sh"))
  k9s_script             = file(abspath("${path.module}/../../../bootstrap/scripts/install-k9s.sh"))
  argocd_password_script = file(abspath("${path.module}/../../../bootstrap/scripts/get-argocd-admin-password.sh"))
  argocd_script          = file(abspath("${path.module}/../../../bootstrap/scripts/install-argocd.sh"))
  root_app_yaml          = file(abspath("${path.module}/../../../gitops/root/root-app.yaml"))

  bootstrap_script = <<-EOT
    #!/usr/bin/env bash
    set -euo pipefail

    exec > >(tee -a /var/log/demo-bootstrap.log) 2>&1

    echo "starting demo bootstrap"

    echo "installing k3s"
    /opt/bootstrap/install-k3s-server.sh

    until test -f /etc/rancher/k3s/k3s.yaml; do
      sleep 2
    done

    mkdir -p /root/.kube
    ln -sf /etc/rancher/k3s/k3s.yaml /root/.kube/config

    echo "installing k9s"
    /opt/bootstrap/install-k9s.sh

    export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
    echo "waiting for k3s api"
    for _ in $(seq 1 60); do
      if kubectl get nodes >/dev/null 2>&1; then
        break
      fi
      sleep 5
    done

    echo "k3s nodes"
    kubectl get nodes

    echo "installing argocd"
    /opt/bootstrap/install-argocd.sh

    echo "applying root app"
    root_app_applied=false
    for _ in $(seq 1 30); do
      if kubectl apply -f /opt/bootstrap/root-app.yaml; then
        echo "root app applied"
        root_app_applied=true
        break
      fi
      echo "root app apply failed, retrying"
      sleep 10
    done

    if [[ "$root_app_applied" != true ]]; then
      echo "root app never applied"
      exit 1
    fi

    echo "bootstrap complete"
  EOT

  user_data = templatefile("${path.module}/cloud-init.yaml.tftpl", {
    k3s_server_script      = local.k3s_server_script
    k9s_script             = local.k9s_script
    argocd_password_script = local.argocd_password_script
    argocd_script          = local.argocd_script
    root_app_yaml          = local.root_app_yaml
    bootstrap_script       = local.bootstrap_script
  })
}

module "vm" {
  source = "../../modules/vm"

  name                = var.name
  region              = var.region
  size                = var.size
  image               = var.image
  ssh_public_key_path = var.ssh_public_key_path
  user_data           = local.user_data
  tag_name            = digitalocean_tag.this.name
}

module "firewall" {
  source = "../../modules/firewall"

  name              = var.name
  allowed_ssh_cidrs = var.allowed_ssh_cidrs
  tag_name          = digitalocean_tag.this.name
}
