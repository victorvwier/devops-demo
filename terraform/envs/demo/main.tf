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
  k3s_server_script = file(abspath("${path.module}/../../../bootstrap/scripts/install-k3s-server.sh"))
  argocd_script     = file(abspath("${path.module}/../../../bootstrap/scripts/install-argocd.sh"))

  bootstrap_script = <<-EOT
    #!/usr/bin/env bash
    set -euo pipefail

    /opt/bootstrap/install-k3s-server.sh

    export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
    for _ in $(seq 1 60); do
      if kubectl get nodes >/dev/null 2>&1; then
        break
      fi
      sleep 5
    done

    kubectl get nodes
    /opt/bootstrap/install-argocd.sh
  EOT

  user_data = templatefile("${path.module}/cloud-init.yaml.tftpl", {
    k3s_server_script = local.k3s_server_script
    argocd_script     = local.argocd_script
    bootstrap_script  = local.bootstrap_script
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
