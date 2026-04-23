variable "name" {
  type    = string
  default = "demo"
}

variable "region" {
  type    = string
  default = "nyc1"
}

variable "size" {
  type    = string
  default = "s-4vcpu-8gb"
}

variable "image" {
  type    = string
  default = "ubuntu-22-04-x64"
}

variable "ssh_public_key_path" {
  type    = string
  default = "~/.ssh/id_rsa.pub"
}

variable "allowed_ssh_cidrs" {
  type    = list(string)
  default = ["0.0.0.0/0"]
}
