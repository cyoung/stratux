#!/bin/bash

fwName=firmware_psRam

RED='\033[1;31m'
NC='\033[0m' # No Color

function cleanup {
  sudo systemctl start stratux
}
trap cleanup EXIT


cd "$(dirname "$0")"

if [ -e /dev/serialin ]; then
  usbDevice=/dev/serialin
else
  echo ""
  echo "To which USB would you like to install the firmware at?"
  echo ""
  echo -e "${RED}If you are unsure about which USB device is which (sorry we cannot detect..) then unplug${NC}"
  echo -e "${RED}the device you do not want to flash and re-run this script.${NC}"
  echo ""
  list=$(find /dev/ -regextype egrep -regex '\/dev\/(ttyACM|ttyAMA|ttyUSB)[0-9]')

  if [ -z "$list" ]; then
      echo "No connected USB, ACM or AMA device found"
      exit
  fi

  select usbDevice in $list
    do test -n "$usbDevice" && break; 
      echo "No USB device selected"
      exit
  done
fi

echo "Installing $fwName to $usbDevice"
echo ""

sudo systemctl stop stratux
echo ""

python3 esptool.py --chip esp32 --port $usbDevice --baud 921600 --before default_reset erase_flash

python3 esptool.py --chip esp32 --port $usbDevice --baud 921600 --before default_reset --after hard_reset \
    write_flash -u --flash_mode dio --flash_freq 40m --flash_size detect \
    0x1000 bootloader_dio_40m.bin 0x10000 $fwName.bin 0x8000 partitions.bin 0x3d0000 spiffs.bin 0xe000 boot_app0.bin
