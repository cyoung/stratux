#!/bin/bash


cd "$(dirname "$0")"

fwfile="esp32-ogn-tracker-bin-10+.zip"
if [ "$1" == "07" ] || [ "$1" == "0.7" ]; then
    fwfile="esp32-ogn-tracker-bin-07.zip"
fi

unzip $fwfile -d /tmp/
systemctl stop stratux
cd /tmp
source flash_USB0.sh
systemctl start stratux


