variable "name" {
  type = string
}

output "name" {
  value = var.name
}

output "ssh_command" {
  value = "ssh ubuntu@<public-ip>"
}
