# -*- mode: ruby -*-
# vi: set ft=ruby :

VAGRANTFILE_API_VERSION = "2"

Vagrant.configure(VAGRANTFILE_API_VERSION) do |config|
  config.vm.box = "ubuntu/trusty64"

  config.vm.provision "shell", privileged: false, inline: <<SCRIPT
sudo apt-get update -y
sudo apt-get install -y mercurial wget git
cd /home/vagrant && wget --quiet --no-check-certificate https://storage.googleapis.com/golang/go1.4.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf /home/vagrant/go1.4.linux-amd64.tar.gz
mkdir /home/vagrant/workspace
echo "export GOPATH=/home/vagrant/workspace" >> .bashrc
echo "export PATH=\\$PATH:/usr/local/go/bin" >> .bashrc
PATH=$PATH:/usr/local/go/bin
SCRIPT

  config.vm.provider "virtualbox" do |v|
    v.memory = 2048
  end
  config.ssh.forward_agent = true

end
