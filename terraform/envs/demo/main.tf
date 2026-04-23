terraform {
  required_version = ">= 1.5.0"
  required_providers {
    local = {
      source  = "hashicorp/local"
      version = "~> 2.5"
    }
  }
}

module "vm" {
  source = "../../modules/vm"

  name = var.name
}

module "firewall" {
  source = "../../modules/firewall"

  allowed_ssh_cidrs = var.allowed_ssh_cidrs
}
