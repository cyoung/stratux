#!/bin/bash


overlayctl unlock
cd /overlay/robase/opt/stratux/mapdata

echo
read -p "Download OpenFlightMaps VFR Charts Europe (~700 MiB)? [y/n]" -n 1 -r
if [[ $REPLY =~ ^[Yy]$ ]]; then
    overlayctl unlock
    wget -N https://abatzill.de/stratux/openflightmaps.mbtiles
    overlayctl lock
fi

echo
read -p "Download US Sectional VFR Charts (~4.9 GiB)? [y/n]" -n 1 -r
if [[ $REPLY =~ ^[Yy]$ ]]; then
    wget -N https://abatzill.de/stratux/vfrsec.mbtiles
fi

overlayctl lock

systemctl restart stratux