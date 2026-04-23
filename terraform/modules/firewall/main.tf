variable "allowed_ssh_cidrs" {
  type = list(string)
}

output "allowed_ssh_cidrs" {
  value = var.allowed_ssh_cidrs
}
