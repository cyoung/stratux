#!/bin/bash

rm -rf /root/stratux-update
mkdir -p /root/stratux-update
cd /root/stratux-update
mv -f /var/log/stratux.sqlite /var/log/stratux.sqlite.`date +%s`
rm -f /var/log/stratux.sqlite-wal /var/log/stratux.sqlite-shm
