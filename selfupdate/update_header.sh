#!/bin/bash

rm -rf /root/stratux-update
mkdir -p /root/stratux-update
cd /root/stratux-update
rm -f /var/log/stratux.sqlite /var/log/stratux.sqlite-wal /var/log/stratux.sqlite-shm
rm -f /var/log/stratux.log
