#!/bin/sh
while true
  do
    sleep 30
    /usr/bin/mlat-client --input-type dump1090 --input-connect localhost:30005 --server feed.adsbexchange.com:31090 --no-udp --results beast,connect,localhost:30104
  done