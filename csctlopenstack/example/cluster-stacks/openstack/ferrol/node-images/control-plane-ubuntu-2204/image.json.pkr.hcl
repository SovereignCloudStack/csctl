packer {
  required_plugins {
    qemu = {
      source  = "github.com/hashicorp/qemu"
      version = "~> 1"
    }
  }
}

locals {
  scripts          = "${path.root}/scripts"
  http_directory   = "${path.root}/http"
}

source "qemu" "ubuntu" {
  accelerator      = "kvm"
  boot_command     = ["c<wait>linux /casper/vmlinuz --- autoinstall ds='nocloud-net;s=http://{{ .HTTPIP }}:{{ .HTTPPort }}'<enter><wait><wait><wait>initrd /casper/initrd<enter><wait><wait><wait>boot<enter>"]
  boot_wait        = "${var.boot_wait}"
  cpu_model        = "host"
  cpus             = "${var.cpus}"
  disk_compression = "${var.disk_compression}"
  disk_discard     = "${var.disk_discard}"
  disk_size        = "${var.disk_size}"
  firmware         = "${var.firmware}"
  format           = "qcow2"
  headless         = "${var.headless}"
  http_directory   = "${local.http_directory}"
  iso_checksum     = "${var.image_checksum}"
  iso_url          = "${var.image_url}"
  memory           = "${var.memory}"
  net_device       = "virtio-net"
  output_directory = "${var.output_directory}"
  qemu_binary      = "${var.qemu_binary}"
  shutdown_command = "echo '${var.ssh_password}' | sudo -S -E sh -c 'usermod -L ${var.ssh_username} && ${var.shutdown_command}'"
  ssh_password     = "${var.ssh_password}"
  ssh_username     = "${var.ssh_username}"
  ssh_wait_timeout = "30m"
  vm_name          = "${var.build_name}"
}

build {
  sources = ["source.qemu.ubuntu"]

  provisioner "shell" {
    inline = ["while [ ! -f /var/lib/cloud/instance/boot-finished ]; do echo 'Waiting for cloud-init...'; sleep 1; done"]
  }

  provisioner "shell" {
    environment_vars = ["PACKER_OS_IMAGE=${var.os}", "PACKER_ARCH=${var.arch}"]
    execute_command  = "echo '${var.ssh_password}' | {{ .Vars }} sudo -E -S bash -x '{{ .Path }}'"
    scripts          = ["${local.scripts}/base.sh", "${local.scripts}/cilium-requirements.sh", "${local.scripts}/cri.sh", "${local.scripts}/kubernetes.sh", "${local.scripts}/cleanup.sh"]
  }

}
