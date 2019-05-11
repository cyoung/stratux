#!/bin/bash

# DO NOT CALL ME DIRECTLY!
# This script is called by mk_europe_edition.sh via qemu

cd /root/stratux
source /root/.bashrc

# Make sure that the upgrade doesn't restart services in the chroot..
mkdir /root/fake
ln -s /bin/true /root/fake/initctl
ln -s /bin/true /root/fake/invoke-rc.d
ln -s /bin/true /root/fake/restart
ln -s /bin/true /root/fake/start
ln -s /bin/true /root/fake/stop
ln -s /bin/true /root/fake/start-stop-daemon
ln -s /bin/true /root/fake/service
ln -s /bin/true /root/fake/deb-systemd-helper

# Fake a proc FS for raspberrypi-sys-mods_20170519_armhf... Extend me as needed
mkdir -p /proc/sys/vm/

apt update
PATH=/root/fake:$PATH apt dist-upgrade --yes
PATH=/root/fake:$PATH apt install --yes libjpeg8-dev libconfig9
apt-get clean

rm -r /proc/*
rm -r /root/fake

# For some reason, qemu build fails unless we use a single compilation thread. Compilation takes quite long...
export GOMAXPROCS=1
go get -u github.com/kidoman/embd/embd
make clean
# Sometimes go build fails for some reason.. we will just try three times and hope for the best
make
make
make
make install
# systemctl daemon-reload doesn't work in qemu because there is no systemd running. Needed?