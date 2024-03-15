
variable "arch" {
  type    = string
  default = "amd64"
}

variable "boot_wait" {
  type    = string
  default = "10s"
}

variable "build_name" {
  type    = string
  default = "control-plane-ubuntu-2204"
}

variable "cpus" {
  type    = string
  default = "2"
}

variable "disk_compression" {
  type    = string
  default = "false"
}

variable "disk_discard" {
  type = string
  default = "unmap"
}

variable "disk_size" {
  type    = string
  default = "20480"
}

variable "firmware" {
  type    = string
  default = ""
}

variable "headless" {
  type    = string
  default = "true"
}

variable "image_checksum" {
  type    = string
  default = "file:http://old-releases.ubuntu.com/releases/22.04/SHA256SUMS"
}

variable "image_url" {
  type    = string
  default = "https://old-releases.ubuntu.com/releases/22.04/ubuntu-22.04.3-live-server-amd64.iso"
}

variable "memory" {
  type    = string
  default = "2048"
}

variable "os" {
  type    = string
  default = "ubuntu-22.04"
}

variable "qemu_binary" {
  type    = string
  default = "qemu-system-x86_64"
}

variable "scripts" {
  type    = string
  default = "{{template_dir}}/scripts"
}

variable "shutdown_command" {
  type    = string
  default = "shutdown -P now"
}

variable "ssh_password" {
  type    = string
  default = "builder"
}

variable "ssh_username" {
  type    = string
  default = "builder"
}
