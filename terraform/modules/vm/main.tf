terraform {
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
  }
}

variable "name" {
  type = string
}

variable "region" {
  type = string
}

variable "size" {
  type = string
}

variable "image" {
  type = string
}

variable "ssh_public_key_path" {
  type = string
}

variable "user_data" {
  type = string
}

locals {
  ssh_public_key = trimspace(file(pathexpand(var.ssh_public_key_path)))
}

resource "digitalocean_ssh_key" "this" {
  name       = "${var.name}-ssh-key"
  public_key = local.ssh_public_key
}

resource "digitalocean_droplet" "this" {
  name      = "${var.name}-node"
  region    = var.region
  size      = var.size
  image     = var.image
  ssh_keys  = [digitalocean_ssh_key.this.id]
  user_data = var.user_data

  tags = [var.name]
}

output "name" {
  value = digitalocean_droplet.this.name
}

output "public_ip" {
  value = digitalocean_droplet.this.ipv4_address
}

output "ssh_command" {
  value = "ssh ubuntu@${digitalocean_droplet.this.ipv4_address}"
}

output "k3s_command" {
  value = "ssh ubuntu@${digitalocean_droplet.this.ipv4_address} 'sudo k3s kubectl get nodes'"
}
