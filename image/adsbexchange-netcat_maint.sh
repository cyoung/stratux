#!/bin/sh
while true
  do
    sleep 30
    /bin/nc 127.0.0.1 30005 | /bin/nc feed.adsbexchange.com 30005
  done