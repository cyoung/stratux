#!/bin/bash

read -t 1 -n 10000 discard # discard input buffer

read -p "Download OpenFlightMaps VFR Charts Europe (~700 MiB)? [y/n]" -n 1 -r
if [[ $REPLY =~ ^[Yy]$ ]]; then
    wget -N https://abatzill.de/stratux/openflightmaps.mbtiles
fi

read -p "Download US Sectional VFR Charts (~4.9 GiB)? [y/n]" -n 1 -r
if [[ $REPLY =~ ^[Yy]$ ]]; then
    wget -N https://abatzill.de/stratux/openflightmaps.mbtiles
fi