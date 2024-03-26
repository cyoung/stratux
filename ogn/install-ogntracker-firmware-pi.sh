#!/bin/bash


cd "$(dirname "$0")"

fwfile="esp32-ogn-tracker-bin-10+.zip"
if [ "$1" == "07" ] || [ "$1" == "0.7" ]; then
    fwfile="esp32-ogn-tracker-bin-07.zip"
fi
if [ "$1" == "sx1262" ]; then
    fwfile="esp32-ogn-tracker-bin-10+-sx1262.zip"
fi

unzip $fwfile -d /tmp/
systemctl stop stratux
cd /tmp

python3 esptool.py --chip esp32 --port /dev/serialin --baud 921600 --before default_reset --after hard_reset \
    write_flash -u --flash_mode dio --flash_freq 40m --flash_size detect 0x1000 \
    build/bootloader/bootloader.bin 0x10000 build/esp32-ogn-tracker.bin 0x8000 build/partitions.bin

systemctl start stratux


