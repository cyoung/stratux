#!/bin/bash


cd "$(dirname "$0")"

fwfile="$1.zip"
echo $fwfile
if [ ! -f $fwfile ]; then
    echo "Unknown firmware $1"
    exit 1
fi

unzip $fwfile -d /tmp/
systemctl stop stratux
cd /tmp

python3 esptool.py --chip esp32 --port /dev/serialin --baud 921600 --before default_reset --after hard_reset \
    write_flash -u --flash_mode dio --flash_freq 40m --flash_size detect 0x1000 \
    bootloader.bin 0x10000 firmware.bin 0x8000 partitions.bin

systemctl start stratux


