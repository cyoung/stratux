#!/bin/bash

# DO NOT CALL BE DIRECTLY!
# This script is called by mk_europe_edition.sh via qemu

cd /root/stratux
source /root/.bashrc

apt install --yes libjpeg8-dev libconfig9

# For some reason, qemu build fails unless we use a single compilation thread. Compilation takes quite long...
export GOMAXPROCS=1
make
make install
# systemctl daemon-reload doesn't work in qemu because there is no systemd running. Needed?