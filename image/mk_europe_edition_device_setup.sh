#!/bin/bash

# DO NOT CALL BE DIRECTLY!
# This script is called by mk_europe_edition.sh via qemu

cd /root/stratux
source /root/.bashrc

apt install --yes libjpeg8-dev libconfig9
make -j 8
make install
systemctl daemon-reload