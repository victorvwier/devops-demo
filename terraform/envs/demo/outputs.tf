output "vm_name" {
  value = module.vm.name
}

output "public_ip" {
  value = module.vm.public_ip
}

output "ssh_command" {
  value = module.vm.ssh_command
}

output "k3s_command" {
  value = module.vm.k3s_command
}
