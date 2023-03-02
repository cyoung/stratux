#!/bin/bash

cd /
overlayctl unlock
cd /overlay/robase/opt/stratux/mapdata


echo "Waiting for time synchronization..."
systemctl start systemd-timesyncd
while [ "$(timedatectl show | grep NTPSync | grep yes)" == "" ]; do
    echo -n "."
    sleep 1
done
systemctl stop systemd-timesyncd




echo
read -p "Download OpenFlightMaps VFR Charts Europe (~700 MiB)? [y/n]" -n 1 -r
if [[ $REPLY =~ ^[Yy]$ ]]; then
    wget -N https://abatzill.de/stratux/openflightmaps.mbtiles
fi

echo
read -p "Download US Sectional VFR Charts (~4.9 GiB)? [y/n]" -n 1 -r
if [[ $REPLY =~ ^[Yy]$ ]]; then
    wget -N https://abatzill.de/stratux/vfrsec.mbtiles
fi

cd /
sync
overlayctl lock

systemctl restart stratux