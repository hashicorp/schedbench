# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure(2) do |config|
  config.vm.box = "puppetlabs/ubuntu-14.04-64-nocm"
  config.vm.provider "vmware_fusion" do |v|
    v.vmx["memsize"] = "8192"
    v.vmx["numvcpus"] = "2"
  end
  config.vm.provision "shell", inline: <<-SHELL
    sudo apt-get update
    sudo apt-get install -y unzip curl
    wget https://releases.hashicorp.com/nomad/0.2.3/nomad_0.2.3_linux_amd64.zip
    sudo unzip -d /usr/local/bin nomad*.zip
    sudo curl -sSL https://get.docker.com/ | sh
    sudo usermod -aG docker vagrant
  SHELL
end
