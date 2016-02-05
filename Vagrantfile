# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure(2) do |config|
  config.vm.box = "puppetlabs/ubuntu-14.04-64-nocm"
  config.vm.provider "vmware_fusion" do |v|
    v.vmx["memsize"] = "8192"
    v.vmx["numvcpus"] = "2"
  end
  config.vm.provision "shell", inline: <<-SHELL
    # Base setup
    sudo apt-get update
    sudo apt-get install -y unzip curl git-core

    # Setup Go
    wget -O /tmp/go.tgz https://storage.googleapis.com/golang/go1.5.3.linux-amd64.tar.gz
    sudo tar -C /usr/local -xf /tmp/go.tgz
    echo "export GOPATH=/root/gopath" | sudo tee -a /root/.profile
    echo "export PATH=$PATH:/usr/local/go/bin:/root/gopath/bin" | sudo tee -a /root/.profile

    # Build Nomad
    sudo -i go get -d github.com/hashicorp/nomad
    sudo -i go build -o /usr/local/bin/nomad github.com/hashicorp/nomad

    # Setup Docker
    sudo curl -sSL https://get.docker.com/ | sh
    sudo usermod -aG docker vagrant

    # Setup Nomad Config
    cat > /tmp/nomad.hcl <<EOF
client {
  node_class = "foobarbaz"
  options = {
    driver.raw_exec.enable = "1"
    docker.cleanup.container = "false"
  }
}
EOF
    sudo mv /tmp/nomad.hcl /usr/local/etc/nomad.hcl

    # Setup Nomad Service
    cat > /tmp/nomad.upstart <<EOF
start on runlevel [2345]
stop on runlevel [!2345]
respawn
script
    exec /usr/local/bin/nomad agent -dev -config /usr/local/etc/nomad.hcl >> /var/log/nomad.log 2>&1
end script
EOF
    sudo mv /tmp/nomad.upstart /etc/init/nomad.conf

    # Start Nomad
    sudo start nomad

    # Get/start Redis
    sudo apt-get -y install redis-server
  SHELL
end
